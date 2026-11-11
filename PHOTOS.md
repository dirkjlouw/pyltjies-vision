# Taking the test photos

About 30 photos is enough to see how well a reader does. More is better.

## Taking them

- Stand the phone where it would stand in a game, roughly in front of the board and a little to one side. Use the normal camera, no zoom.
- Get the whole board in, including the numbers.
- Throw three darts, take the photo, write down the darts, pull them out, repeat.
- Mix it up: some easy darts in big singles, some on or near wires, a few in trebles, doubles and the bull, a couple in one bed. Include some with darts hiding each other.
- Take a few in the light you usually play in, and a few in worse light.
- Leave a dart that's outside the double ring in and write `M` for it. A dart that bounced out isn't in the photo, so leave it out.

iPhone photos are fine as HEIC or JPEG.

## Labelling them

Put the photos in `photos/` and list them in `photos/labels.csv`, one line per photo: the file name, a comma, then the darts in the board, in any order.

```csv
file,darts
IMG_0001.jpg,T20 S5 S1
IMG_0002.heic,D16 25
IMG_0003.jpg,S19 S19 T19
```

Write darts as `S20`, `D20`, `T20`, `25` for the outer bull, `50` for the inner bull, and `M` for outside the double ring. A plain `20` means a single.

Score each dart where its tip went in, as you would in a game. If you're not sure, pull it out slowly and look before you write it down.

`photos/` is git-ignored, so your photos stay on your computer. See [photos/labels.example.csv](photos/labels.example.csv).
