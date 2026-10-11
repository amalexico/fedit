export function debounce1(fn, wait) {
  let timer = null;
  return (...args) => {
    clearTimeout(timer);
    timer = setTimeout(() => fn(...args), wait);
  };
}

export function throttle1(fn, limit) {
  let inFlight = false;
  return (...args) => {
    if (inFlight) return;
    inFlight = true;
    fn(...args);
    setTimeout(() => { inFlight = false; }, limit);
  };
}

export function memoize1(fn) {
  const cache = new Map();
  return (arg) => {
    if (cache.has(arg)) return cache.get(arg);
    const result = fn(arg);
    cache.set(arg, result);
    return result;
  };
}
