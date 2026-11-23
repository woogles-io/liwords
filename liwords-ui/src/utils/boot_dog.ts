// The loading dog (index.html) sits in #root until the app's first render
// wipes it, and can be petted: a press squishes it, its handlers marking it
// "petted" while held and stamping each press and release in data-pet-at, by
// performance.now(): the page's own clock, which setting the system clock
// does not move, and which the app reads too.
// Someone petting it holds the app off: the app draws once the dog is let go
// and has been left alone for petLinger ms, so the petting can go on and no
// press lands on the app. A dog never petted, or none, lets it draw at once.
export const petLinger = 1000;

export const drawWhenDogLeftAlone = (draw: () => void, linger = petLinger) => {
  const dog = document.querySelector<HTMLElement>(".boot-splash img");
  const listening: Array<[EventTarget, string, () => void]> = [];
  const listen = (target: EventTarget, type: string, fn: () => void) => {
    target.addEventListener(type, fn);
    listening.push([target, type, fn]);
  };
  let timer: ReturnType<typeof setTimeout> | undefined;
  let drawn = false;
  const held = () => !!dog?.classList.contains("petted");

  const check = () => {
    clearTimeout(timer);
    // a held dog waits for its release, which checks again
    if (drawn || held()) return;
    // the last pet's stamp, none if the dog was never petted
    const petAt = dog?.dataset.petAt;
    const wait = petAt == null ? 0 : Number(petAt) + linger - performance.now();
    if (wait > 0) {
      timer = setTimeout(check, wait);
      return;
    }
    drawn = true;
    listening.forEach(([target, type, fn]) =>
      target.removeEventListener(type, fn),
    );
    draw();
  };

  // A press whose release never comes (the page left mid-press) counts as
  // let go when the page loses focus or is hidden.
  const away = () => {
    if (dog && held()) {
      dog.classList.remove("petted");
      dog.dataset.petAt = String(performance.now());
    }
    check();
  };

  if (dog) {
    // after the dog's own handlers, which come first
    for (const type of ["pointerup", "pointercancel", "pointerleave"]) {
      listen(dog, type, check);
    }
  }
  listen(window, "blur", away);
  listen(document, "visibilitychange", () => {
    if (document.visibilityState === "hidden") away();
  });
  check();
};
