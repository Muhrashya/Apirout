# 42: Global Toast Notification Temporal Deduplication

**What to build:** Adds a 1,000ms temporal window debounce deduplication layer to `ToastProvider` in the Web Console. When identical notifications with matching `type` and `message` are dispatched in rapid succession (e.g. from rapid double-clicks or multiple event listeners), duplicates within 1 second are suppressed, ensuring only one clean toast pill appears.

**Blocked by:** 41

**Status:** completed

- [x] `ToastProvider` in `web/src/components/Toast.tsx` maintains a ref of recent toast dispatches with timestamps
- [x] Multiple calls to `showToast` with identical `type` and `message` within 1,000ms do not spawn duplicate toast pills
- [x] Distinct toast messages or differing toast types continue to dispatch immediately without delay
- [x] Typechecks cleanly with `npm run build` in `web/`
