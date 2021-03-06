# Based on http://github.com/f-secure-foundry/tamago-example

BUILD_USER = $(shell whoami)
BUILD_HOST = $(shell hostname)
BUILD_DATE = $(shell /bin/date -u "+%Y-%m-%d %H:%M:%S")
BUILD = ${BUILD_USER}@${BUILD_HOST} on ${BUILD_DATE}
REV = $(shell git rev-parse --short HEAD 2> /dev/null)

APP := wallera
TARGET ?= "usbarmory"
GOENV := GO_EXTLINK_ENABLED=0 CGO_ENABLED=0 GOOS=tamago GOARM=7 GOARCH=arm
TEXT_START := 0x80010000 # ramStart (defined in imx6/imx6ul/memory.go) + 0x10000
LDFLAGS = -s -w -T $(TEXT_START) -E _rt0_arm_tamago -R 0x1000 -X 'main.Build=${BUILD}' -X 'main.Revision=${REV}'
DEBUG_TAG = "debug"
GOFLAGS = -tags ${TARGET},${DEBUG_TAG} -ldflags "${LDFLAGS}"
SHELL = /bin/bash

.PHONY: clean install test wallera-linux setup-wallera-linux

#### primary targets ####

all: $(APP)

imx: $(APP).imx

imx_signed: $(APP)-signed.imx

elf: $(APP)

#### utilities ####

check_tamago:
	@if [ "${TAMAGO}" == "" ] || [ ! -f "${TAMAGO}" ]; then \
		echo 'You need to set the TAMAGO variable to a compiled version of https://github.com/f-secure-foundry/tamago-go'; \
		exit 1; \
	fi

check_usbarmory_git:
	@if [ "${USBARMORY_GIT}" == "" ]; then \
		echo 'You need to set the USBARMORY_GIT variable to the path of a clone of'; \
		echo '  https://github.com/f-secure-foundry/usbarmory'; \
		exit 1; \
	fi

check_hab_keys:
	@if [ "${HAB_KEYS}" == "" ]; then \
		echo 'You need to set the HAB_KEYS variable to the path of secure boot keys'; \
		echo 'See https://github.com/f-secure-foundry/usbarmory/wiki/Secure-boot-(Mk-II)'; \
		exit 1; \
	fi

dcd:
	@if test "${TARGET}" = "usbarmory"; then \
		cp -f $(GOMODCACHE)/$(TAMAGO_PKG)/board/f-secure/usbarmory/mark-two/imximage.cfg $(APP).dcd; \
	elif test "${TARGET}" = "mx6ullevk"; then \
		cp -f $(GOMODCACHE)/$(TAMAGO_PKG)/board/nxp/mx6ullevk/imximage.cfg $(APP).dcd; \
	else \
		echo "invalid target - options are: usbarmory, mx6ullevk"; \
		exit 1; \
	fi

clean: tee_clean
	rm -fr $(APP) $(APP).bin $(APP).imx $(APP)-signed.imx $(APP).csf $(APP).dcd $(APP)-linux

install: $(APP)
	@ssh usbarmory@10.0.0.1 sudo rm /boot/tamago
	@scp $(APP) usbarmory@10.0.0.1:/boot/tamago
	@ssh usbarmory@10.0.0.1 sudo reboot

wallera-linux:
	$(TAMAGO) build -gcflags "all=-N -l" -o ./wallera-linux ./cmd/wallera-linux 

setup-wallera-linux: wallera-linux
	@echo "You will be prompted for your root password, because we have to load some kernel modules and setup permissions"
	sudo bash cmd/wallera-linux/load_kernel_modules.sh
	sudo ./wallera-linux -setup
	sudo chown $$USER /dev/hidg0

tee_demo:
	$(MAKE) -C tee nonsecure_demo_os 
	$(MAKE) -C tee cryptography_applet 
	$(MAKE) -C tee trusted_os
	mv tee/bin/trusted_os.imx tee_demo.imx

tee_demo_qemu: tee_demo
	$(MAKE) -C tee qemu

tee_clean: 
	$(MAKE) -C tee clean

tee_wallera: wallera-tee 
	$(MAKE) -C tee cryptography_applet
	cp wallera tee/trusted_os/assets/nonsecure_os.elf
	$(MAKE) -C tee trusted_os
	mv tee/bin/trusted_os.imx tee_wallera.imx

#### dependencies ####
$(APP): check_tamago
	$(GOENV) $(TAMAGO) build ${GOFLAGS} -o ${APP} ./firmware/

$(APP)-tee: TEXT_START=0x80010000
$(APP)-tee: TARGET:=$(addsuffix  ,"tee_enabled",$(TARGET))
$(APP)-tee: check_tamago
	$(GOENV) $(TAMAGO) build ${GOFLAGS} -o ${APP} ./firmware/

test: check_tamago
	$(TAMAGO) test $(shell go list ./... | sed -E '/(wallera|firmware|cmd|cert|certs)$$/d')

$(APP).dcd: check_tamago
$(APP).dcd: GOMODCACHE=$(shell ${TAMAGO} env GOMODCACHE)
$(APP).dcd: TAMAGO_PKG=$(shell grep "github.com/f-secure-foundry/tamago v" go.mod | awk '{print $$1"@"$$2}')
$(APP).dcd: dcd

$(APP).bin: $(APP)
	$(CROSS_COMPILE)objcopy -j .text -j .rodata -j .shstrtab -j .typelink \
	    -j .itablink -j .gopclntab -j .go.buildinfo -j .noptrdata -j .data \
	    -j .bss --set-section-flags .bss=alloc,load,contents \
	    -j .noptrbss --set-section-flags .noptrbss=alloc,load,contents\
	    $(APP) -O binary $(APP).bin

$(APP).imx: check_usbarmory_git $(APP).bin $(APP).dcd
	mkimage -n $(APP).dcd -T imximage -e $(TEXT_START) -d $(APP).bin $(APP).imx
	# Copy entry point from ELF file
	dd if=$(APP) of=$(APP).imx bs=1 count=4 skip=24 seek=4 conv=notrunc

#### secure boot ####

$(APP)-signed.imx: check_usbarmory_git check_hab_keys $(APP).imx
	${USBARMORY_GIT}/software/secure_boot/usbarmory_csftool \
		--csf_key ${HAB_KEYS}/CSF_1_key.pem \
		--csf_crt ${HAB_KEYS}/CSF_1_crt.pem \
		--img_key ${HAB_KEYS}/IMG_1_key.pem \
		--img_crt ${HAB_KEYS}/IMG_1_crt.pem \
		--table   ${HAB_KEYS}/SRK_1_2_3_4_table.bin \
		--index   1 \
		--image   $(APP).imx \
		--output  $(APP).csf && \
	cat $(APP).imx $(APP).csf > $(APP)-signed.imx
