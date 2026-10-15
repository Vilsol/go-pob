# The Linux packaging host for machines without the WebKit2GTK 4.1 / GTK 3 development packages (`task package-desktop`
# runs package.sh in it); CI runs package.sh on an ubuntu-24.04 runner with the same packages instead.
FROM ubuntu:24.04
RUN apt-get update \
	&& DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
		build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.1-dev glib-networking libglib2.0-bin \
		ca-certificates curl file \
	&& rm -rf /var/lib/apt/lists/*
