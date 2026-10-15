Unicode true

## The go-pob installer, built by ../package.sh:
##   makensis -DARG_WAILS_AMD64_BINARY=<go-pob.exe> -DINFO_PRODUCTVERSION=<x.y.z> -DOUTFILE=<installer.exe> [-DSIGN_CMD=<cmd>] project.nsi
## A per-user install (%LOCALAPPDATA%\Programs\go-pob, no UAC prompt, the updater can replace the exe), the pob:// link
## handler, and go-pob as an "Open with" choice for .xml files (never the default .xml program). wails_tools.nsh is
## Wails' own (`wails3 generate build-assets`), used for its architecture check, WebView2 bootstrapper and uninstall key.

!define INFO_PROJECTNAME "go-pob"
!define INFO_COMPANYNAME "go-pob"
!define INFO_PRODUCTNAME "go-pob"
!define INFO_COPYRIGHT "MIT"
!define WAILS_INSTALL_SCOPE "user"
!define REQUEST_EXECUTION_LEVEL "user"
!define PROGID "go-pob.build"

!include "wails_tools.nsh"

VIProductVersion "${INFO_PRODUCTVERSION}.0"
VIFileVersion    "${INFO_PRODUCTVERSION}.0"

VIAddVersionKey "CompanyName"     "${INFO_COMPANYNAME}"
VIAddVersionKey "FileDescription" "${INFO_PRODUCTNAME} Installer"
VIAddVersionKey "ProductVersion"  "${INFO_PRODUCTVERSION}"
VIAddVersionKey "FileVersion"     "${INFO_PRODUCTVERSION}"
VIAddVersionKey "LegalCopyright"  "${INFO_COPYRIGHT}"
VIAddVersionKey "ProductName"     "${INFO_PRODUCTNAME}"

ManifestDPIAware true

!include "MUI.nsh"

!define MUI_ICON "..\icon.ico"
!define MUI_UNICON "..\icon.ico"
!define MUI_ABORTWARNING
!define MUI_FINISHPAGE_RUN "$INSTDIR\${PRODUCT_EXECUTABLE}"

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "English"

!ifdef SIGN_CMD
!uninstfinalize '${SIGN_CMD} "%1"' = 0
!endif

Name "${INFO_PRODUCTNAME}"
OutFile "${OUTFILE}"
InstallDir "$LOCALAPPDATA\Programs\${INFO_PRODUCTNAME}"
ShowInstDetails show

Function .onInit
    !insertmacro wails.checkArchitecture
FunctionEnd

Section
    !insertmacro wails.setShellContext
    !insertmacro wails.webview2runtime

    SetOutPath $INSTDIR
    !insertmacro wails.files

    CreateShortcut "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"
    CreateShortCut "$DESKTOP\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"

    DeleteRegKey HKCU "Software\Classes\pob"
    WriteRegStr HKCU "Software\Classes\pob" "" "URL:Path of Building build link"
    WriteRegStr HKCU "Software\Classes\pob" "URL Protocol" ""
    WriteRegStr HKCU "Software\Classes\pob\DefaultIcon" "" "$INSTDIR\${PRODUCT_EXECUTABLE},0"
    WriteRegStr HKCU "Software\Classes\pob\shell\open\command" "" '"$INSTDIR\${PRODUCT_EXECUTABLE}" "%1"'

    WriteRegStr HKCU "Software\Classes\${PROGID}" "" "Path of Building build"
    WriteRegStr HKCU "Software\Classes\${PROGID}\DefaultIcon" "" "$INSTDIR\${PRODUCT_EXECUTABLE},0"
    WriteRegStr HKCU "Software\Classes\${PROGID}\shell\open\command" "" '"$INSTDIR\${PRODUCT_EXECUTABLE}" "%1"'
    WriteRegStr HKCU "Software\Classes\.xml\OpenWithProgids" "${PROGID}" ""
    System::Call 'shell32::SHChangeNotify(i 0x08000000, i 0, p 0, p 0)'

    !insertmacro wails.writeUninstaller
SectionEnd

Section "uninstall"
    !insertmacro wails.setShellContext

    RMDir /r "$AppData\${PRODUCT_EXECUTABLE}" # the WebView2 data folder; builds and settings stay in %APPDATA%\go-pob
    RMDir /r $INSTDIR

    Delete "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk"
    Delete "$DESKTOP\${INFO_PRODUCTNAME}.lnk"

    DeleteRegKey HKCU "Software\Classes\pob"
    DeleteRegValue HKCU "Software\Classes\.xml\OpenWithProgids" "${PROGID}"
    DeleteRegKey HKCU "Software\Classes\${PROGID}"
    System::Call 'shell32::SHChangeNotify(i 0x08000000, i 0, p 0, p 0)'

    !insertmacro wails.deleteUninstaller
SectionEnd
