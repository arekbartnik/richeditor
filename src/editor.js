import { Command, moveToDocSide, setTextblockType, unwrapBlock } from "wordgard/command";
import { serialize } from "wordgard/doc";
import { Dialog, InputRule, KeyBinding, Wordgard } from "wordgard/editor";
import { history, redoButton, undoButton } from "wordgard/history";
import { phrases } from "wordgard/phrases";
import { blockDoc, blockquote, bulletList, emphasis, lineBreak, link, orderedList, paragraph, strong, underline } from "wordgard/schema";
import { GardState } from "wordgard/state";
import { Emphasis, Heading, Link, Paragraph, Strong, Underline } from "wordgard/types";

const base = [blockDoc(), paragraph(), lineBreak(), history()];

// linkTarget turns what a user types into a link address. A bare domain gets
// https://, and schemes other than web, mail and phone are refused.
const linkTarget = (input) => {
  const url = input.trim();
  if (!url || /\s/.test(url)) return null;
  if (/^(https?:|mailto:|tel:|\/|#)/i.test(url)) return url;
  if (/^[a-z][a-z\d+.-]*:(?!\d)/i.test(url)) return null;
  return url.includes("@") && !url.includes("/") ? `mailto:${url}` : `https://${url}`;
};

// Adapted from Wordgard's toggleLink, MIT licensed, to word the prompt from
// the button and to check the address.
const linkFeature = (el) => {
  const toggleLink = (wg) => {
    if (wg.state.readOnly) return false;
    const open = Dialog.get(wg, toggleLink);
    if (open) {
      open.close();
      return true;
    }
    const { selection, doc } = wg.state;
    if (selection.empty) return false;
    const remove = [];
    for (const { from, to } of selection.ranges)
      doc.iterate(from, to, (node, pos) => {
        const has = Link.isInSet(node.marks);
        if (has) remove.push({ from: pos, to: pos + node.length, remove: has });
      });
    if (remove.length) {
      wg.dispatch({ changes: remove, userEvent: "mark.remove" });
      return true;
    }
    Dialog.show(wg, {
      key: toggleLink,
      label: el.dataset.urlLabel,
      input: { type: "text", name: "url" },
      submitLabel: el.dataset.submitLabel,
      class: "wg-link-dialog",
      focus: true,
      closedBy: (tr) => !!tr.selection,
      tooltip: { pos: selection.from },
    }).result.then((form) => {
      wg.focus();
      const url = form && linkTarget(form.elements.namedItem("url")?.value ?? "");
      if (url) wg.dispatch({ changes: wg.state.selection.ranges.map((r) => ({ from: r.from, to: r.to, add: Link.of(url) })), userEvent: "mark.add" });
    });
    return true;
  };
  return {
    config: [
      GardState.schemaElement.of(Link),
      KeyBinding.of({ key: "Mod-k", run: toggleLink }),
      link.tooltip,
      link.pasteOver,
      phrases.translatePartial({ dialog_close: el.dataset.closeLabel }),
    ],
    button: { run: toggleLink, active: link.button.active, enable: link.button.enable },
  };
};

const clearFormatting =
  (marks) =>
  ({ state }) => {
    const { selection, sel } = state;
    const block = sel.head.textblockParent;
    const range = selection.empty && block ? { from: block.start, to: block.end } : { from: selection.from, to: selection.to };
    let tr = state.update({ changes: marks.map((remove) => ({ ...range, remove })) });
    let changes = tr.changes;
    for (let spec; (spec = unwrapBlock({ state: tr.state }, null)); ) {
      tr = tr.state.update(spec);
      if (tr.changes.empty) break;
      changes = changes.compose(tr.changes);
    }
    const plain = setTextblockType({ state: tr.state }, Paragraph);
    if (plain) {
      tr = tr.state.update(plain);
      changes = changes.compose(tr.changes);
    }
    return { changes, selection: tr.state.selection, scrollIntoView: true, userEvent: "format.clear" };
  };

// selectionIn is Wordgard's own check for its textblock style buttons, which
// it does not export.
const selectionIn = (tag) => (state) => {
  const block = state.sel.head.textblockParent;
  return !!block && block.start == state.sel.anchor.textblockParent?.start && block.node.tag.eq(tag);
};

// textStyleFeature drives the text style dropdown. Its option values are 0
// for a paragraph and the level for a heading, and only the offered levels
// get a key binding and a "## " input rule.
const textStyleFeature = (el) => {
  const levels = [...el.options].map((o) => Number(o.value)).filter((level) => level > 0);
  const styles = [["0", Paragraph], ...levels.map((level) => [String(level), Heading.of(level)])];
  const set = (tag) => (target) => setTextblockType(target, tag);
  return {
    config: [
      levels.length ? GardState.schemaElement.of(Heading) : [],
      KeyBinding.of({ key: "Ctrl-Shift-0", run: set(Paragraph) }),
      levels.map((level) => [
        KeyBinding.of({ key: `Ctrl-Shift-${level}`, run: set(Heading.of(level)) }),
        InputRule.textblockType(new RegExp(`^#{${level}} $`), Heading.of(level), true),
      ]),
    ],
    control: {
      sync(state) {
        el.value = styles.find(([, tag]) => selectionIn(tag)(state))?.[0] ?? "";
        el.disabled = state.readOnly;
      },
      change(wg) {
        const tag = styles.find(([value]) => value === el.value)?.[1];
        if (tag) Command.dispatch(wg, set(tag));
        wg.focus();
      },
    },
  };
};

// Each button brings its formatting into the schema, so an editor without
// a button cannot hold that formatting at all.
const features = {
  undo: () => ({ button: undoButton }),
  redo: () => ({ button: redoButton }),
  strong: () => ({ config: strong(), button: strong.button, mark: Strong }),
  emphasis: () => ({ config: emphasis(), button: emphasis.button, mark: Emphasis }),
  underline: () => ({ config: underline(), button: underline.button, mark: Underline }),
  link: linkFeature,
  bulletList: () => ({ config: bulletList(), button: bulletList.toggleButton }),
  orderedList: () => ({ config: orderedList(), button: orderedList.toggleButton }),
  blockquote: () => ({ config: blockquote(), button: blockquote.button }),
  textStyle: textStyleFeature,
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

// A dropdown has to take focus to open, so only buttons leave it in the text.
const keepFocus = (evt) => evt.target.closest("button[data-command]") && evt.preventDefault();

class RichEditor extends HTMLElement {
  static formAssociated = true;

  #internals = this.attachInternals();
  #wg = null;
  #html = null;
  #message = "";
  #toolbar = [];
  #controls = [];
  #config = [];
  #popover = null;

  connectedCallback() {
    // A value set before the element upgraded shadows the setter.
    if (Object.hasOwn(this, "value")) {
      const html = this.value;
      delete this.value;
      this.#html = html;
    }
    const els = [...this.querySelectorAll("[data-command]")];
    const enabled = els.map((el) => [el, features[el.dataset.command]?.(el) ?? {}]);
    const marks = enabled.flatMap(([, f]) => f.mark ?? []);
    this.#toolbar = enabled.map(([el, f]) => [el, el.dataset.command === "clear" ? { run: clearFormatting(marks) } : f.button]);
    this.#controls = enabled.flatMap(([el, f]) => (f.control ? [[el, f.control]] : []));
    this.#config = [base, enabled.map(([, f]) => f.config ?? [])];
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
    this.addEventListener("change", this.#changeControl);
    // A moved editor reconnects with the text typed so far.
    this.#mount(this.#html ?? this.getAttribute("value") ?? "");
  }

  disconnectedCallback() {
    this.#popover.destroy();
    this.removeEventListener("mousedown", keepFocus);
    this.removeEventListener("click", this.#runCommand);
    this.removeEventListener("change", this.#changeControl);
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
    const button = this.#toolbar.find(([b]) => b === el)?.[1];
    if (!button?.run) return;
    Command.dispatch(this.#wg, button.run);
    // The link prompt takes focus inside the editor and keeps it.
    if (!this.#wg.dom.contains(document.activeElement)) this.#wg.focus();
  };

  #changeControl = (evt) => {
    const control = this.#controls.find(([el]) => el === evt.target)?.[1];
    if (!control) return;
    // The dropdown has no name, so its change is no change of the form.
    evt.stopPropagation();
    control.change(this.#wg);
  };

  #syncToolbar(state) {
    for (const [el, button] of this.#toolbar) {
      if (button?.active) el.setAttribute("aria-pressed", String(button.active(state)));
      if (button?.enable) el.disabled = !button.enable(state);
    }
    for (const [, control] of this.#controls) control.sync(state);
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
        this.#config,
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
