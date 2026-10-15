# A clean Ubuntu 24.04 desktop with only what go-pob's AppImage asks of the host (WebKit2GTK 4.1, GTK 3, CA certificates),
# plus a private X server, a session bus and xdg-utils for the headless runs; the deb installs its own dependencies.
FROM ubuntu:24.04
RUN apt-get update \
	&& DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
		libwebkit2gtk-4.1-0 libgtk-3-0t64 ca-certificates xvfb dbus xdg-utils desktop-file-utils procps libglib2.0-bin \
	&& rm -rf /var/lib/apt/lists/*
