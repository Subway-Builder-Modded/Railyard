# Installing Railyard on macOS

Railyard is not currently signed with an Apple Developer certificate, so macOS Gatekeeper will block it by default. Follow the steps below to install and run it.

## Install from DMG

1. Download the latest `railyard-macos-universal.dmg` from the [Releases](https://github.com/Subway-Builder-Modded/Railyard/releases) page.
2. Open the DMG and drag **Railyard** into your **Applications** folder.

## First launch

When you first try to open Railyard, macOS will show a dialog saying the app "can't be opened because Apple cannot check it for malicious software." This is expected for unsigned apps.

### Option A: Right-click to open (recommended)

1. Open **Finder** and navigate to **Applications**.
2. **Right-click** (or Control-click) on **Railyard**.
3. Select **Open** from the context menu.
4. A dialog will appear with an **Open** button — click it.

You only need to do this once. After the first launch, you can open Railyard normally.

### Option B: System Settings

1. Try to open Railyard normally (it will be blocked).
2. Open **System Settings** > **Privacy & Security**.
3. Scroll down to the **Security** section. You should see a message about Railyard being blocked.
4. Click **Open Anyway** and confirm.

### Option C: Terminal

If neither option above works, you can remove the quarantine attribute:

```bash
xattr -cr /Applications/railyard.app
```

Then open Railyard normally.

## Install from ZIP

If you prefer not to use the DMG:

1. Download `railyard-macos-universal.zip` from the [Releases](https://github.com/Subway-Builder-Modded/Railyard/releases) page.
2. Extract the ZIP (double-click it in Finder).
3. Move `railyard.app` to your **Applications** folder.
4. Follow the **First launch** steps above.

## Uninstalling

Drag **Railyard** from Applications to the Trash. To remove app data:

```bash
rm -rf ~/Library/Application\ Support/railyard
```
