import { Command, moveToDocSide, unwrapBlock } from "wordgard/command";
import { serialize } from "wordgard/doc";
import { Wordgard } from "wordgard/editor";
import { history } from "wordgard/history";
import { blockDoc, blockquote, bulletList, emphasis, lineBreak, orderedList, paragraph, strong } from "wordgard/schema";
import { Emphasis, Strong } from "wordgard/types";

const config = [blockDoc(), paragraph(), lineBreak(), strong(), emphasis(), bulletList(), orderedList(), blockquote(), history()];

const clearFormatting = ({ state }) => {
  const { selection, sel } = state;
  const block = sel.head.textblockParent;
  const range = selection.empty && block ? { from: block.start, to: block.end } : { from: selection.from, to: selection.to };
  let tr = state.update({ changes: [Strong, Emphasis].map((remove) => ({ ...range, remove })) });
  let changes = tr.changes;
  for (let spec; (spec = unwrapBlock({ state: tr.state }, null)); ) {
    tr = tr.state.update(spec);
    if (tr.changes.empty) break;
    changes = changes.compose(tr.changes);
  }
  return { changes, selection: tr.state.selection, scrollIntoView: true, userEvent: "format.clear" };
};

const unwrapTightListParagraphs = (root) => {
  for (const list of root.querySelectorAll("ul, ol")) {
    const items = [...list.children];
    if (items.some((item) => item.querySelectorAll(":scope > p").length > 1)) continue;
    for (const item of items) for (const p of item.querySelectorAll(":scope > p")) p.replaceWith(...p.childNodes);
  }
};

const toHTML = (doc) => {
  const box = document.createElement("div");
  box.append(serialize(doc).toDOM());
  unwrapTightListParagraphs(box);
  return box.innerHTML;
};

const buttons = {
  strong: strong.button,
  emphasis: emphasis.button,
  bulletList: bulletList.toggleButton,
  orderedList: orderedList.toggleButton,
  blockquote: blockquote.button,
  clear: { run: clearFormatting },
};

const selectionPopover = (host, { content, focus, selectedRect }) => {
  const popover = host.querySelector(".rich-editor-popover");
  if (!popover) return { sync() {}, destroy() {} };
  let frame = 0;
  const open = () => popover.matches(":popover-open");
  const place = () => {
    if (popover.contains(document.activeElement)) return;
    const rect = content().contains(document.activeElement) ? selectedRect() : null;
    if (!rect) {
      if (open()) popover.hidePopover();
      return;
    }
    if (!open()) popover.showPopover();
    const gap = 8;
    const { width, height } = popover.getBoundingClientRect();
    // Touch screens show the system's copy menu above a selection.
    const below = matchMedia("(pointer: coarse)").matches || rect.top - height - gap < 0;
    popover.style.top = `${below ? rect.bottom + gap : rect.top - height - gap}px`;
    popover.style.left = `${Math.max(gap, Math.min((rect.left + rect.right - width) / 2, innerWidth - width - gap))}px`;
  };
  const sync = () => {
    cancelAnimationFrame(frame);
    frame = requestAnimationFrame(place);
  };
  const onKeydown = (evt) => {
    if (evt.altKey && evt.key === "F10" && open()) {
      evt.preventDefault();
      popover.querySelector("button").focus();
    } else if (evt.key === "Escape" && popover.contains(evt.target)) {
      evt.stopPropagation();
      focus();
    }
  };
  const onFocusout = (evt) => host.contains(evt.relatedTarget) || sync();
  host.addEventListener("keydown", onKeydown);
  host.addEventListener("focusout", onFocusout);
  addEventListener("scroll", sync, true);
  addEventListener("resize", sync);
  return {
    sync,
    destroy() {
      cancelAnimationFrame(frame);
      host.removeEventListener("keydown", onKeydown);
      host.removeEventListener("focusout", onFocusout);
      removeEventListener("scroll", sync, true);
      removeEventListener("resize", sync);
      if (open()) popover.hidePopover();
    },
  };
};

const keepFocus = (evt) => evt.target.closest("[data-command]") && evt.preventDefault();

class RichEditor extends HTMLElement {
  static formAssociated = true;

  #internals = this.attachInternals();
  #wg = null;
  #html = null;
  #message = "";
  #toolbar = [];
  #popover = null;

  connectedCallback() {
    // A value set before the element upgraded shadows the setter.
    if (Object.hasOwn(this, "value")) {
      const html = this.value;
      delete this.value;
      this.#html = html;
    }
    this.#toolbar = [...this.querySelectorAll("[data-command]")].map((el) => [el, buttons[el.dataset.command]]);
    this.#popover = selectionPopover(this, {
      content: () => this.#wg.contentDOM,
      focus: () => this.#wg.focus(),
      selectedRect: () => {
        const { selection } = this.#wg.state;
        if (selection.empty) return null;
        const start = this.#wg.coordsAtPos(selection.from, 1);
        const end = this.#wg.coordsAtPos(selection.to, -1);
        return { top: Math.min(start.top, end.top), bottom: Math.max(start.bottom, end.bottom), left: Math.min(start.left, end.left), right: Math.max(start.right, end.right) };
      },
    });
    this.addEventListener("mousedown", keepFocus);
    this.addEventListener("click", this.#runCommand);
    // A moved editor reconnects with the text typed so far.
    this.#mount(this.#html ?? this.getAttribute("value") ?? "");
  }

  disconnectedCallback() {
    this.#popover.destroy();
    this.removeEventListener("mousedown", keepFocus);
    this.removeEventListener("click", this.#runCommand);
    this.#wg.dom.remove();
    this.#wg = null;
  }

  formResetCallback() {
    this.value = this.getAttribute("value") ?? "";
  }

  get value() {
    return this.#html;
  }

  set value(html) {
    if (this.#wg) this.#mount(html);
    else this.#html = html;
  }

  get form() {
    return this.#internals.form;
  }

  get validity() {
    return this.#internals.validity;
  }

  get validationMessage() {
    return this.#internals.validationMessage;
  }

  get textLength() {
    return this.#wg ? [...this.#wg.state.doc.textContent({ blockSeparator: "" })].length : 0;
  }

  checkValidity() {
    return this.#internals.checkValidity();
  }

  reportValidity() {
    return this.#internals.reportValidity();
  }

  setCustomValidity(message) {
    this.#message = message;
    this.#validate();
  }

  focus() {
    this.#wg?.focus();
  }

  focusEnd() {
    this.#wg.focus();
    Command.dispatch(this.#wg, moveToDocSide, { side: "end" });
  }

  #runCommand = (evt) => {
    const el = evt.target.closest("[data-command]");
    if (!el) return;
    Command.dispatch(this.#wg, buttons[el.dataset.command].run);
    this.#wg.focus();
  };

  #syncToolbar(state) {
    for (const [el, button] of this.#toolbar) if (button.active) el.setAttribute("aria-pressed", String(button.active(state)));
  }

  #validate() {
    if (this.#wg) this.#internals.setValidity(this.#message ? { customError: true } : {}, this.#message, this.#wg.contentDOM);
  }

  #mount(html) {
    this.#wg?.dom.remove();
    this.#html = html;
    this.#internals.setFormValue(html);
    const contentClass = this.getAttribute("content-class");
    this.#wg = Wordgard.create({
      doc: html,
      parent: this,
      config: [
        config,
        Wordgard.label(this.getAttribute("label") ?? ""),
        Wordgard.editorAttributes.of({ class: "rich-editor-content" }),
        contentClass ? Wordgard.contentAttributes.of({ class: contentClass }) : [],
        Wordgard.updateListener.of((update) => {
          this.#syncToolbar(update.state);
          this.#popover.sync();
          if (!update.docChanged) return;
          this.#html = toHTML(update.state.doc);
          this.#internals.setFormValue(this.#html);
          this.dispatchEvent(new Event("input", { bubbles: true }));
        }),
      ],
    });
    this.#validate();
    this.#syncToolbar(this.#wg.state);
  }
}

customElements.define("rich-editor", RichEditor);
