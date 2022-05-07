# Sets up dependencies and functions
# sudo nix-shell https://github.com/sgillespie/nixos-yubikey-luks/archive/master.tar.gz

SLOT=2
# yubikey setup if necessary
# ykpersonalize -"$SLOT" -ochal-resp -ochal-hmac

SALT_LENGTH=16
salt="$(dd if=/dev/random bs=1 count=$SALT_LENGTH 2>/dev/null | rbtohex)"
echo "salt calculated"

challenge="$(echo -n $salt | openssl dgst -binary -sha512 | rbtohex)"
echo "challenge: $challenge"

response="$(ykchalresp -2 -x $challenge 2>/dev/null)"
echo "response: $response"


parted -s /dev/vda -- mklabel gpt

parted -s /dev/vda -- mkpart ESP fat32 4MiB 512MiB
parted -s /dev/vda -- set 1 esp 

parted -s /dev/vda -- mkpart primary 512MiB 100%
echo "partitioned"


EFI_PART=/dev/vda1
LUKS_PART=/dev/vda2


mkfs.vfat -F 32 -n NIXBOOT "$EFI_PART"

EFI_MNT=/root/boot
mkdir -p "$EFI_MNT"
mount "$EFI_PART" "$EFI_MNT"

echo "NIXBOOT partitioned and mounted"

KEY_LENGTH=512
ITERATIONS=1000

k_user=pass
k_luks="$(echo -n $k_user | pbkdf2-sha512 $(($KEY_LENGTH / 8)) $ITERATIONS $response | rbtohex)"

# if no user password
# k_luks="$(echo | pbkdf2-sha512 $(($KEY_LENGTH / 8)) $ITERATIONS $response | rbtohex)"

STORAGE=/crypt-storage/default
mkdir -p "$(dirname $EFI_MNT$STORAGE)"

echo -ne "$salt\n$ITERATIONS" > $EFI_MNT$STORAGE

umount /root/boot

CIPHER=aes-xts-plain64
HASH=sha512
echo -n "$k_luks" | hextorb | cryptsetup luksFormat --cipher="$CIPHER" --key-size="$KEY_LENGTH" --hash="$HASH" --key-file=- "$LUKS_PART"

LUKSROOT=NIXROOT
echo -n "$k_luks" | hextorb | cryptsetup luksOpen $LUKS_PART $LUKSROOT --key-file=-

echo "luks setup"

mkfs.ext4 /dev/mapper/$LUKSROOT

echo "second formated"

mount /dev/mapper/$LUKSROOT /mnt

mkdir -p /mnt/boot

mount $EFI_PART /mnt/boot

# In hardware-configuration.nix
# hardware config
boot.initrd.kernelModules = [ "vfat" "nls_cp437" "nls_iso8859-1" "usbhid" ];

# Enable support for the YubiKey PBA
boot.initrd.luks.yubikeySupport = true;

# Configuration to use your Luks device
boot.initrd.luks.devices = {
  "NIXROOT" = {
    device = "/dev/vda2";
    preLVM = true; # You may want to set this to false if you need to start a network service first
    yubikey = {
      slot = 2;
      twoFactor = true; # Set to false if you did not set up a user password.
      storage = {
        device = "/dev/vda1";
      };
    };
  }; 
};