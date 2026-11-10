PKG_CONFIG_PATH := $(CURDIR)/pkgconfig:$(PKG_CONFIG_PATH)
export PKG_CONFIG_PATH
export CGO_ENABLED := 1

.PHONY: build test install systemd-install

build:
	go build -o techtips .

test:
	go test ./...

install: build
	install -Dm755 techtips $(HOME)/.local/bin/techtips
	install -Dm644 .env.example $(HOME)/.config/techtips/.env.example
	@test -f $(HOME)/.config/techtips/.env || cp .env.example $(HOME)/.config/techtips/.env
	mkdir -p $(HOME)/.local/share/techtips

systemd-install: install
	install -Dm644 systemd/techtips.service $(HOME)/.config/systemd/user/techtips.service
	systemctl --user daemon-reload
	systemctl --user disable --now techtips.timer 2>/dev/null || true
	systemctl --user enable --now techtips.service
