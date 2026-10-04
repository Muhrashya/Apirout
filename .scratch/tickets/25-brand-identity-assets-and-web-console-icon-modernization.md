# 25: Brand Identity Assets & Web Console Icon Modernization

**What to build:** Modernize Apirout's visual brand identity by replacing the AI-generated ribbon logo with a clean, high-impact gradient lightning bolt on a transparent background, ensuring crisp visual rendering across GitHub documentation, dark/light modes, and the Web Console application.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [ ] Create a high-resolution, transparent background PNG logo featuring a minimalist sharp lightning bolt with a warm orange-to-coral linear gradient (`#FF8A00` to `#FF3E3E`), matching the reference icon.
- [ ] Save the high-resolution logo to `docs/logo.png` (512x512).
- [ ] Save the application icon to `web/public/logo.png` (128x128) and update browser tab icon `web/public/favicon.png` (64x64).
- [ ] Verify that the new logo renders seamlessly without dark borders or artifacts in both GitHub light/dark markdown modes.
- [ ] Build the frontend (`npm run build`) and verify via browser that the new icon displays cleanly in the Web Console navigation bar and login views on `http://localhost:8002`.
