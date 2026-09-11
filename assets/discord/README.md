# Discord branding

Images for the bot's application in the [Discord Developer Portal](https://discord.com/developers/applications).
They are uploaded by hand; nothing in the build reads them.

| File | Portal setting | Size |
|---|---|---|
| `avatar-1024.png` | **General Information › App Icon** and **Bot › Icon** | 1024×1024 |
| `banner-1360x480.png` | **Bot › Banner** | 1360×480 |

The `.svg` files are the masters; the PNGs are renders of them at the sizes above.

## avatar

Fita, the desktop UI's app icon (`assets/icon/` in `discord_instants_player_ui`): an ink cassette with
mustard reels on enamel blue, with the same geometry and colours. Unlike the UI's masters it is a full
square with no platform mask, because Discord crops avatars to a circle itself. The cassette clears
that circle.

## banner

The Esmalte palette's dark ground with its six instant card fills laid out as a favourites grid. The
mustard card is the one "playing", so it shows stop instead of play. It is twice Discord's 680×240
profile size, and the lower-left third is deliberately empty: Discord overlaps the avatar there.

## Colours

All from Esmalte, converted from the design system's oklch values.

| Role | Hex |
|---|---|
| Enamel blue (avatar ground, card) | `#0d66b4` |
| Ink (cassette, banner ground) | `#13181d` |
| Mustard (reels, playing card) | `#d89e09` |
| Card fills | `#a83231` `#257b51` `#008a90` `#8c4083` |
| Glyph ink on dark cards | `#f5f8fc` |

If Fita or Esmalte changes in the UI, update the SVGs to match and re-render the PNGs at the sizes above.
