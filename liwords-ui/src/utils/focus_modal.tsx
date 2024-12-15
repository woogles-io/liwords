import React, { useState } from "react";
import { Modal as AntdModal } from "antd";

// XXX: previously some Modal came from here instead:
// import Modal from 'antd/lib/modal/Modal';
// No idea what the differences are.

// This Modal focuses on the first focusable element.

// Usage: Just import this Modal instead of Antd's.

// Implementation note: It adds a div since AntdModal does not expose ref=.

export const Modal = (props: React.ComponentProps<typeof AntdModal>) => {
  const [divElt, setDivElt] = useState<HTMLDivElement | null>(null);
  const shownDivElt = props.visible ? divElt : null;

  React.useEffect(() => {
    if (shownDivElt) {
      // When hiding a modal and then showing it again, ref is already set.
      // Need setTimeout to let the modal opening animation complete first.
      const t = setTimeout(() => {
        const allChildren = shownDivElt.querySelectorAll<HTMLElement>("*");

        // Negative tabIndex should not be selected.
        // Zero tabIndex (default) are selected after positive tabIndex.
        // https://developer.mozilla.org/en-US/docs/Web/API/HTMLElement/tabIndex
        const candidates = [];
        for (let i = 0; i < allChildren.length; ++i) {
          let { tabIndex } = allChildren[i];
          if (tabIndex < 0) continue;
          if (tabIndex === 0) tabIndex = Infinity;
          candidates.push({ tabIndex, i });
        }

        // Non-negative tabIndex should be selected in order.
        candidates.sort((a, b) => {
          if (a.tabIndex < b.tabIndex) return -1;
          if (a.tabIndex > b.tabIndex) return 1;
          if (a.i < b.i) return -1;
          if (a.i > b.i) return 1;
          return 0;
        });

        // Try to .focus() until it works. It should no-op if it doesn't work.
        // This handles disabled, input[type=hidden], display: none, etc.
        for (const { i } of candidates) {
          const child = allChildren[i];
          child.focus();
          if (document.activeElement === child) break;
        }
      }, 0);
      return () => {
        clearTimeout(t);
      };
    }
  }, [shownDivElt]);

  return (
    <AntdModal {...props}>
      <div ref={setDivElt}>{props.children}</div>
    </AntdModal>
  );
};
