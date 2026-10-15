# go-pob desktop

go-pob as a desktop app: the same app as [pob.gg](https://pob.gg), in a window of its own, with builds kept in a folder
on disk (Path of Building's own Builds folder when one is found), `pob://` links and build files opening in it, and
game data cached for offline use. Downloads are on the repository's Releases page.

| system | download | needs |
|---|---|---|
| Windows 10 / 11 (x64) | `go-pob-amd64-installer.exe`, or `go-pob-windows-amd64.zip` (portable) | the WebView2 runtime, part of Windows 11 and of Windows 10 since 2021; the installer fetches it when it is missing |
| macOS 12 or later | `go-pob-macos-arm64.dmg` (Apple Silicon), `go-pob-macos-amd64.dmg` (Intel) | — |
| Linux (x86_64) | `go-pob-x86_64.AppImage`, `.deb`, `.rpm` | WebKit2GTK 4.1 and GTK 3, glibc 2.39 or later (Ubuntu 24.04, Fedora 40 and later) |

## First launch

The Windows and macOS builds are not yet signed by a certificate the operating system knows, so each warns once.

### Windows

1. Run the installer. SmartScreen shows **Windows protected your PC**: click **More info**, then **Run anyway**.
2. The installer needs no administrator rights: it installs for your user into `%LOCALAPPDATA%\Programs\go-pob`, adds
   Start menu and desktop shortcuts, makes `pob://` links open go-pob, and adds go-pob to **Open with** for `.xml`
   files (your default `.xml` program is not changed). Uninstall from **Settings › Apps**.

For the portable zip: right-click the zip, **Properties**, tick **Unblock**, **OK**, then extract it; the first start of
`go-pob.exe` may still show the SmartScreen prompt above. The portable exe registers nothing: `pob://` links need the
installer.

### macOS

1. Open the dmg and drag **go-pob** onto **Applications**.
2. Open go-pob. macOS says it cannot check the app for malicious software and offers **Done** or **Move to Trash**: click
   **Done**.
3. Open **System Settings › Privacy & Security**, scroll to **Security**, and click **Open Anyway** next to the line
   about go-pob, then confirm with **Open Anyway** and your password. (On macOS 14 and earlier, Control-click the app in
   Applications and choose **Open** instead.)

Or, in Terminal, remove the download's quarantine flag before the first open:

```shell
xattr -dr com.apple.quarantine /Applications/go-pob.app
```

`pob://` links open go-pob after its first launch. go-pob is offered under **Open With** for
`.xml` files without becoming their default app.

### Linux

Make the AppImage executable (`chmod +x go-pob-x86_64.AppImage`) and run it; **Help › Add to applications menu** adds
a menu entry and the `pob://` handler. The deb and rpm do both on install.

## Updates

**Check for update** (Ctrl+U, or Settings) asks the release page for a newer version and accepts only one signed with
the project's key. The Windows exe (portable or installed), the macOS app and the plain Linux binary download it and
restart into it when their folder is writable; the AppImage, deb and rpm, and a copy in a folder you cannot write
(Program Files, an app run from its dmg), name the new version and its release page instead.

## Where things are kept

| | Windows | macOS | Linux |
|---|---|---|---|
| Path of Building's Builds folder (used when found; its **Options › Build save path** wins) | `Documents\Path of Building\Builds` (the Documents folder Windows reports, OneDrive included) | — | inside Wine / Proton / Lutris / Bottles prefixes, `~/Documents/Path of Building/Builds` |
| builds (when no Path of Building Builds folder is found; **File › Builds folder…** picks another) | `%APPDATA%\go-pob\builds` | `~/Library/Application Support/go-pob/builds` | `~/.config/go-pob/builds` |
| settings | `%APPDATA%\go-pob\settings` | `~/Library/Application Support/go-pob/settings` | `~/.config/go-pob/settings` |
| game data cache | `%LOCALAPPDATA%\go-pob\webview` | `~/Library/Caches/go-pob/webview` | `~/.cache/go-pob/webview` |

Uninstalling leaves builds and settings in place.
