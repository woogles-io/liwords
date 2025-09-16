import json
import math
import os
import qrcode
import sys

try:
    # Debug info
    print("LD_LIBRARY_PATH:", os.environ.get("LD_LIBRARY_PATH"))
    print("FONTCONFIG_PATH:", os.environ.get("FONTCONFIG_PATH"))
    print("Python path:", sys.path)

    # Try loading cairo
    import cairo

    print("Cairo version:", cairo.version)
except Exception as e:
    import traceback

    error_details = traceback.format_exc()
    print(f"ERROR: {error_details}")


import cairo
import webcolors
import requests
from io import BytesIO


class URLNotUniqueException(Exception):
    pass


# format color
def format_color(*color, **kwargs):  # color is a tuple
    alpha = 1 if "alpha" not in kwargs else kwargs["alpha"]
    if isinstance(color[0], str):
        color = color[0].strip().lower()
        if color[0] == "#":
            # assume hex
            rgb = list(webcolors.hex_to_rgb(color))
        else:
            # assume color name
            rgb = list(webcolors.name_to_rgb(color))
    elif isinstance(color, (list, tuple)):
        rgb = color[:3]  # gets first three values in list
        if len(color) == 4:
            alpha = color[3]
        too_big = [x for x in rgb if x > 1]
        if not too_big:
            return color
    else:
        rgb = [0, 0, 0]  # sets default color to black

    normalized_rgb = [x / 255 for x in rgb]
    normalized_rgb.append(alpha)

    return normalized_rgb


def to_spread(i):
    if i < 0:
        return str(i)
    return "+" + str(i)


def wl(standing):
    wins = standing.get("wins", 0) + standing.get("draws", 0) / 2
    losses = standing.get("losses", 0) + standing.get("draws", 0) / 2
    if int(wins) == wins:
        wins = int(wins)
    if int(losses) == losses:
        losses = int(losses)
    return str(wins), str(losses)


def create_simple_qr(data, error_correction=qrcode.constants.ERROR_CORRECT_L):
    # Instantiate QRCode object with desired settings
    qr = qrcode.QRCode(
        error_correction=error_correction,
        box_size=10,
        border=4,
    )
    # Add data to the QR code
    qr.add_data(data)
    best_fit_version = qr.best_fit()

    if best_fit_version <= 3:
        qr.box_size = 10
    elif best_fit_version == 4:
        qr.box_size = 9
    elif best_fit_version == 5:
        qr.box_size = 8
    # version 5 with error correct L can hold up to 154 characters. we really
    # don't need any more.
    else:
        raise Exception("Too much data for QR code")

    qr.make(fit=True)  # Fit to the smallest possible QR code version

    # Create an image from the QR Code instance
    img = qr.make_image(fill_color="black", back_color="white")

    return img


class ScorecardCreator:
    def __init__(
        self, tourney, show_opponents: bool, show_seeds: bool, show_qrcode: bool
    ):
        self.tourney = tourney
        self.show_opponents = show_opponents
        self.show_seeds = show_seeds
        self.show_qrcode = show_qrcode
        self.url_uniqueness_trunc = 2
        self.qrcode_urls = set()

    def reset(self):
        self.qrcode_urls = set()

    def set_output_path(self, path):
        self.output_path = path

    def place_qr_code(self, ctx, url):
        self.qrcode_urls.add(url)
        qrcode_img = create_simple_qr(url)
        # Convert QR code to 'RGBA' format
        qr_rgba = qrcode_img.convert("RGBA")

        qr_bytes = bytearray(qr_rgba.tobytes())

        qrsurface = cairo.ImageSurface.create_for_data(
            qr_bytes,
            cairo.FORMAT_ARGB32,
            qr_rgba.width,
            qr_rgba.height,
            qr_rgba.width * 4,
        )

        ctx.save()
        ctx.translate(490, 6)
        ctx.scale(0.20, 0.20)

        ctx.set_source_surface(qrsurface, 0, 0)
        ctx.paint()
        ctx.restore()

    def draw_name_and_tourney_header(
        self, ctx, player, pidx, tourney_name, tourney_logo
    ):
        player_name = player["id"].split(":")[1]
        player_rating = player.get("rating", 0)

        black = format_color("black")
        ctx.new_path()
        ctx.set_font_size(20)

        player_name_x = 25
        if self.show_seeds:
            # circle with number
            ctx.arc(50, 50, 25, 0, 2 * math.pi)
            ctx.set_line_width(1)
            ctx.set_source_rgba(*black)
            ctx.stroke()

            xidx = 45
            if len(str(pidx + 1)) > 1:
                xidx = 40
            ctx.move_to(xidx, 56)
            ctx.show_text(str(pidx + 1))
            player_name_x = 80

        # Show tournament name
        ctx.set_font_size(12)
        ctx.move_to(player_name_x, 76)
        ctx.show_text(tourney_name)

        if self.show_qrcode and not tourney_logo:
            ctx.move_to(300, 75)
            ctx.set_font_size(12)
            ctx.show_text("Enter scores and view standings:")

        if tourney_logo:
            print("Using tourney logo:", tourney_logo)
            if not hasattr(self, "cached_logo"):
                response = requests.get(tourney_logo)
                # Check PNG signature bytes
                is_png = response.content.startswith(b"\x89PNG\r\n\x1a\n")
                if not is_png:
                    print(f"Logo is not a PNG file. URL: {tourney_logo}")
                    try:
                        from PIL import Image

                        img = Image.open(BytesIO(response.content))

                        # Resize to reasonable dimensions to avoid memory issues
                        img.thumbnail((300, 300))
                        output = BytesIO()
                        img.save(output, format="PNG")
                        output.seek(0)

                        self.cached_logo = cairo.ImageSurface.create_from_png(output)
                        print(f"Converted image to PNG")
                    except Exception as e:
                        print(f"Error processing non-PNG logo: {str(e)}")
                        self.cached_logo = None
                else:
                    # It's a PNG, proceed with caution
                    try:
                        # Set a reasonable size limit (e.g., 1MB)
                        if len(response.content) > 1024 * 1024:
                            print(
                                f"Logo too large ({len(response.content)} bytes), skipping"
                            )
                        else:
                            self.cached_logo = cairo.ImageSurface.create_from_png(
                                BytesIO(response.content)
                            )
                    except Exception as e:
                        print(f"Error loading PNG: {str(e)}")
                        self.cached_logo = None

            logo_surface = self.cached_logo

            # Only draw logo if we successfully loaded it
            if logo_surface is not None:
                # Calculate safe width to avoid QR code overlap
                logo_height = logo_surface.get_height()
                logo_width = logo_surface.get_width()

                # QR code starts at x=490, logo starts at x=375
                # Safe width = space between logo start and QR code start = 490-375 = 115
                max_safe_width = 105  # Leave 10pt margin

                # Calculate scale based on both height and width constraints
                height_scale_factor = 70 / logo_height
                width_scale_factor = max_safe_width / logo_width

                # Use the smaller scale factor to ensure logo fits in both dimensions
                scale_factor = min(height_scale_factor, width_scale_factor)

                ctx.save()
                ctx.translate(375, 10)
                ctx.scale(scale_factor, scale_factor)
                ctx.set_source_surface(logo_surface, 0, 0)
                ctx.paint()
                ctx.restore()
            else:
                print("Skipping logo drawing due to previous errors")

        # line for player and name
        ctx.set_font_size(20)
        ctx.move_to(player_name_x, 56)
        ctx.show_text(f"{player_name}  ({player_rating})")
        ctx.set_font_size(12)

    def draw_known_pairings(self, ctx, div, pidx, rect_ht, nrounds, offset, fields):
        ctx.set_font_size(12)
        pid = div["players"]["persons"][pidx]["id"]
        for k, v in div["pairing_map"].items():
            if pidx in v["players"]:
                rd = v.get("round", 0)
                if rd - offset >= 16 or rd - offset < 0:
                    continue
                first = True
                opp = None
                opp_name = None
                for place, pairedidx in enumerate(v["players"]):
                    if pairedidx != pidx:
                        opp = pairedidx
                        opp_name = div["players"]["persons"][opp]["id"].split(":")[1]
                        if place == 0:
                            first = False
                if opp is None:
                    # self-pairing
                    opp = pidx
                    opp_name = v["outcomes"][0].title()
                rdY = 125 + ((rd - offset) * rect_ht) - (2 if nrounds == 8 else 0)
                rdX = 97
                if len(str(opp + 1)) > 1:
                    rdX = 93
                if self.show_seeds:
                    ctx.move_to(rdX, rdY)
                    ctx.show_text(str(opp + 1))
                    ctx.move_to(140, rdY)
                else:
                    ctx.move_to(rdX, rdY)
                ctx.show_text(opp_name)
                # Circle 1st or 2nd
                y = 130 + ((rd - offset) * rect_ht) + (5 if nrounds == 7 else 0) - 2
                ctx.new_sub_path()
                if first:
                    ctx.arc(40, y, 8, 0, 2 * math.pi)
                else:
                    ctx.arc(70, y, 8, 0, 2 * math.pi)
                ctx.stroke()
                # Show scores if they exist
                last_field = len(fields)
                theirscorey = y - 8
                ourscorey = y - 8
                if rd != 0:
                    theirscorey = y - 15
                if len(v.get("outcomes")) == 2 and v["outcomes"][0] in (
                    "LOSS",
                    "WIN",
                    "DRAW",
                ):
                    ctx.move_to(fields[last_field - 3][0] + 20, ourscorey)
                    myscore = v["games"][0]["scores"][0]
                    theirscore = v["games"][0]["scores"][1]
                    if not first:
                        myscore, theirscore = theirscore, myscore
                    ctx.show_text(str(myscore))

                    ctx.move_to(fields[last_field - 2][0] + 20, theirscorey)
                    ctx.show_text(str(theirscore))
                    ctx.move_to(fields[last_field - 1][0] + 10, theirscorey)
                    ctx.show_text(to_spread(myscore - theirscore))
                    # Get cumulative spread and record from standings object.
                    starr = div["standings"][str(rd)]["standings"]
                    for st in starr:
                        if st["player_id"] != pid:
                            continue
                        if rd > 0:
                            ctx.move_to(fields[last_field - 1][0] + 10, y + 5)
                            ctx.show_text(to_spread(st.get("spread", 0)))
                        wins, losses = wl(st)
                        ctx.move_to(fields[last_field - 5][0] + 5, ourscorey)
                        ctx.show_text(wins)
                        ctx.move_to(fields[last_field - 4][0] + 5, ourscorey)
                        ctx.show_text(losses)

        ctx.new_path()

    def draw_row(self, ctx, i, rect_ht, nrounds, fields, offset):
        yi = i - offset

        ctx.rectangle(25, 100 + (yi * rect_ht), 535, rect_ht)
        ctx.stroke()
        if self.show_seeds:
            ctx.arc(  # circle around player's number
                100,
                100 + (yi * rect_ht) + (rect_ht / 2),
                (rect_ht - 4) / 2,
                0,
                2 * math.pi,
            )
            ctx.stroke()

        # Round number
        ctx.set_font_size(18)
        if (i + 1) >= 10:  # If it's two digits move it left a lil.
            ctx.move_to(45, 125 + (yi * rect_ht))
        else:
            ctx.move_to(50, 125 + (yi * rect_ht))
        ctx.show_text(str(i + 1))
        ctx.move_to(35, 130 + (yi * rect_ht) + (5 if nrounds == 7 else 0))
        ctx.set_font_size(8)
        ctx.show_text("1st        2nd")

        last_field = len(fields)
        if i != 0:
            last_field = len(fields) - 1

        for f in fields[1:last_field]:
            ctx.move_to(f[0] - 5, 100 + (yi * rect_ht))  # 340 to 380 at end
            ctx.line_to(f[0] - 5, 100 + (yi * rect_ht) + rect_ht)
            ctx.stroke()
        # Deal with spread box.
        if i > 0:
            f = fields[last_field]
            ctx.move_to(f[0] - 5, 100 + (yi * rect_ht))
            ctx.line_to(f[0] - 5, 100 + (yi * rect_ht) + rect_ht / 2)
            ctx.stroke()
            ctx.move_to(
                fields[last_field - 1][0] - 5, 100 + (yi * rect_ht) + rect_ht / 2
            )
            ctx.line_to(560, 100 + (yi * rect_ht) + rect_ht / 2)
            ctx.stroke()
            ctx.move_to(
                fields[last_field - 1][0] - 2, 100 + (yi * rect_ht) + 7 * (rect_ht / 8)
            )
            ctx.set_font_size(12)
            ctx.show_text("Cumulative:")

    def gen_single_player_scorecard(self, ctx, div, nrounds, meta, pidx, surface):

        drawing_pages = True
        offset = 0
        rounds = list(range(nrounds))

        while drawing_pages:
            player = div["players"]["persons"][pidx]
            if self.show_qrcode:
                idtrunc = player["id"][: self.url_uniqueness_trunc]
                qrcode_url = (
                    f"https://woogles.io{meta['metadata']['slug']}?es={idtrunc}"
                )
                if offset == 0 and qrcode_url in self.qrcode_urls:
                    # If we are in the middle of drawing a multipage scoresheet
                    # don't check for uniqueness after every page.
                    raise URLNotUniqueException()
                self.place_qr_code(ctx, qrcode_url)
            self.draw_name_and_tourney_header(
                ctx,
                player,
                pidx,
                meta["metadata"]["name"],
                meta["metadata"].get("logo", ""),
            )

            # header row
            ctx.rectangle(25, 80, 535, 20)
            ctx.stroke()

            fields = [
                (35, "Round"),
                (85, "Opponent"),
                (300, "Won"),
                (335, "Lost"),
                (370, "Your Score"),
                (440, "Opp Score"),
                (510, "Spread"),
            ]

            for field in fields:
                ctx.move_to(field[0], 95)
                ctx.show_text(field[1])

            # header lines
            for f in fields[1:]:
                ctx.move_to(f[0] - 5, 80)
                ctx.line_to(f[0] - 5, 100)

            rect_ht = 40
            if nrounds == 8:
                # Special case; max 8 rounds for drawing two scorecards on a single sheet.
                rect_ht = 35

            for i in rounds[:16]:
                self.draw_row(ctx, i, rect_ht, nrounds, fields, offset)
            # Draw known pairings
            if self.show_opponents:
                self.draw_known_pairings(
                    ctx, div, pidx, rect_ht, nrounds, offset, fields
                )

            offset += 16
            rounds = rounds[16:]
            if len(rounds) == 0:
                drawing_pages = False
            else:
                surface.show_page()  # New page

    def gen_scorecard(self, surface, ctx, div, nrounds, meta, p1, p2):
        if p1 != p2:
            for idx, pidx in enumerate([p1, p2]):
                ctx.save()
                ctx.translate(0, idx * 396)
                self.gen_single_player_scorecard(ctx, div, nrounds, meta, pidx, surface)
                ctx.restore()
            surface.show_page()  # Add a new page for the next scorecard
        else:
            self.gen_single_player_scorecard(ctx, div, nrounds, meta, p1, surface)
            surface.show_page()  # Add a new page for the next scorecard

        surface.flush()

    def _gen_scorecards(self):
        for divname, div in self.tourney["t"]["divisions"].items():
            nrounds = len(div["round_controls"])
            skip = 2 if nrounds <= 8 else 1
            fname = os.path.join(self.output_path, f"{divname}_scorecards.pdf")
            # 8.5 x 11 inches in points (612 x 792)
            surface = cairo.PDFSurface(fname, 612, 792)
            ctx = cairo.Context(surface)

            ctx.select_font_face(
                "Noto Sans", cairo.FONT_SLANT_NORMAL, cairo.FONT_WEIGHT_NORMAL
            )
            for i in range(0, len(div["players"]["persons"]), skip):
                end = i + skip - 1
                if end > len(div["players"]["persons"]) - 1:
                    end = i
                self.gen_scorecard(
                    surface,
                    ctx,
                    div,
                    nrounds,
                    self.tourney["meta"],
                    i,
                    end,
                )
            surface.finish()

    def gen_scorecards(self):
        success = False
        while success is False:
            try:
                self._gen_scorecards()
            except URLNotUniqueException:
                self.reset()
                self.url_uniqueness_trunc += 1
                print(
                    "Could not create unique URL, trying new trunc length",
                    self.url_uniqueness_trunc,
                )
            else:
                success = True
