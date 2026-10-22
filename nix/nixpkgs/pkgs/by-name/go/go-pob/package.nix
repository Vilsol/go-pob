{
  lib,
  buildGoModule,
  fetchFromGitHub,
  nodejs,
  pnpm_10,
  fetchPnpmDeps,
  pnpmConfigHook,
  pkg-config,
  wrapGAppsHook3,
  gtk3,
  webkitgtk_4_1,
  libsoup_3,
  glib-networking,
  nix-update-script,
}:

buildGoModule (finalAttrs: {
  pname = "go-pob";
  version = "0.1.0";

  src = fetchFromGitHub {
    owner = "Vilsol";
    repo = "go-pob";
    tag = "v${finalAttrs.version}";
    hash = lib.fakeHash;
  };

  # desktop/ is its own module with a `replace` to the repository root, which also holds the wasm
  modRoot = "desktop";
  proxyVendor = true;
  vendorHash = "sha256-OB+gUquyp0r/8lWZ4bYBKYp9Lv7KhsvcTWsVI6WlHXk=";

  overrideModAttrs = {
    dontPnpmConfigure = true;
    preBuild = null;
    # the root module's own requirements (the wasm build), which desktop's build list does not reach
    postBuild = ''
      (cd .. && go mod download)
    '';
  };

  pnpmDeps = fetchPnpmDeps {
    inherit (finalAttrs) pname version src;
    sourceRoot = "${finalAttrs.src.name}/frontend";
    pnpm = pnpm_10;
    fetcherVersion = 4;
    hash = "sha256-00n2kf465A5S3mFaCiQJ4hqOYhswVded1VOPscEcvCc=";
  };
  # relative to modRoot, where configurePhase leaves the build
  pnpmRoot = "../frontend";

  nativeBuildInputs = [
    nodejs
    pnpm_10
    pnpmConfigHook
    pkg-config
    wrapGAppsHook3
  ];

  buildInputs = [
    gtk3
    webkitgtk_4_1
    libsoup_3
    glib-networking
  ];

  # the frontend as desktop/main.go embeds it (VITE_DESKTOP=1, go-pob.wasm.gz only)
  preBuild = ''
    pushd ..
    CGO_ENABLED=0 bash desktop/scripts/wasm.sh
    (cd frontend && VITE_DESKTOP=1 pnpm exec vite build --logLevel warn)
    cp -r frontend/build/. desktop/dist/
    find desktop/dist \( -name go-pob.wasm -o -name go-pob.wasm.br -o -name service-worker.js -o -name _headers \) -delete
    popd
  '';

  subPackages = [ "." ];
  tags = [
    "gtk3"
    "production"
  ];
  ldflags = [
    "-s"
    "-w"
    "-X main.version=${finalAttrs.version}"
    # the store is read-only: no self-update, the page says updates come from Nix
    "-X main.updateManager=Nix"
  ];

  postInstall = ''
    mv $out/bin/desktop $out/bin/go-pob
    install -Dm644 build/linux/go-pob.desktop -t $out/share/applications
    install -Dm644 icon.png $out/share/icons/hicolor/512x512/apps/go-pob.png
  '';

  passthru.updateScript = nix-update-script { };

  meta = {
    description = "Path of Building, ported to Go, as a desktop app";
    homepage = "https://github.com/Vilsol/go-pob";
    license = with lib.licenses; [
      gpl3Only
      mit
    ];
    mainProgram = "go-pob";
    maintainers = with lib.maintainers; [ vilsol ];
    platforms = lib.platforms.linux;
  };
})
