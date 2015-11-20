# happy-tree

This project was inspired by [this video](https://www.youtube.com/watch?v=_DpzAvb3Vk4),
which you should watch first in order to really understand what's going on.

My inspiration came from his noting that happification could be done on numbers
in bases other than 10. I immediately thought of hexadecimal, base-16, since I'm
a programmer and that's what I think of. I also was trying to think of how one
would graphically represent a large happification tree, when I realized that
hexadecimal numbers are colors, and colors graphically represent things nicely!

## Colors

Colors to computers are represented using 3-bytes, encompassing red, green, and
blue. Each byte is represented by two hexadecimal digits, and they are appended
together. For example `FF0000` represents maximum red (`FF`) added to no green
and no blue. `FF5500` represents maximum red (`FF`), some green (`55) and no
blue (`00`), which when added together results in kind of an orange color.

## Happifying colors

In base 10, happifying a number is done by splitting its digits, squaring each
one individually, and adding the resulting numbers. The principal works the same
for hexadecimal numbers:

```
A4F
A*A + 4*4 + F*F
64 + 10 + E1
155 // 341 in decimal
```

So if all colors are 6-digit hexadecimal numbers, they can be happified easily!

```
FF5500
F*F + F*F + 5*5 + 5*5 + 0*0 + 0*0
E1 + E1 + 19 + 19 + 0 + 0
0001F4
```

So `FF5500` (and orangish color) happifies to `0001F4` (A darker blue). Since
order of digits doesn't matter, `5F50F0` also happifies to `0001F4`. From this
fact, we can make a tree (hence the happification tree). I can do this process
on every color from `000000` (black) to `FFFFFF` (white), so I will!

## Representing the tree

So I know I can represent the tree using color, but there's more to decide on
than that. The easy way to represent a tree would be to simply draw a literal
tree graph, with a circle for each color and lines pointing to its parent and
children. But this is boring, and also if I want to represent *all* colors the
resulting image would be enormous and/or unreadable.

I decided on using a hollow, multi-level pie-chart. Using the example
of `000002`, it would look something like this:

![An example of a partial multi-level pie chart](/img/partial.png)

The inner arc represents the color `000002`. The second arc represents the 15
different colors which happify into `000002`, each of them may also have their
own outer arc of numbers which happify to them, and so on.

This representation is nice because a) It looks cool and b) it allows the
melancoils of the hexadecimals to be placed around the happification tree
(numbers which happify into `000001`), which is convenient. It's also somewhat
easier to code than a circle/branch based tree diagram.

An important feature I had to implement was proportional slice sizes. If I were
to give each child of a color an equal size on that arc's edge the image would simply not work.
Some branches of the tree are
extremely deep, while others are very shallow. If all were given the same space,
those deep branches wouldn't even be representable by a single pixel's width,
and would simply fail to show up. So I implemented proportional slice sizes,
where the size of every slice is determined to be proportional to how many total
(recursively) children it has. You can see this in the above example, where the
second level arc is largely comprised of one giant slice, with many smaller
slices taking up the end.

## First attempt

My first attempt resulted in this image (click for 5000x5000 version):

[![Result of first attempt](/img/happy-tree-blue-small.png)](/img/happy-tree-blue.png)

The first thing you'll notice is that it looks pretty neat.

The second thing you'll notice is that there's actually only one melancoil in
the 6-digit hexadecimal number set. The innermost black circle is `000000` which
only happifies to itself, and nothing else will happify to it (sad `000000`).
The second circle represents `000001`, and all of its runty children. And
finally the melancoil, comprised of: `00000D`, `0000A9`, `0000B5`, `000092`,
`000055`, and `000032`.

The final thing you'll notice (or maybe it was the first, since it's really
obvious) is that it's very blue. Non-blue colors are really only represented as
leaves on their trees and don't ever really have any children of their own, so
the blue and black sections take up vastly more space.

This makes sense. The number which should generate the largest happification
result, `FFFFFF`, only results in `000546`, which is primarily blue. So in effect
all colors happify to some shade of blue.

This might have been it, technically this is the happification tree and the
melancoil of 6 digit hexadecimal numbers represented as colors. But it's also
boring, and I wanted to do better.

## Second attempt

The root of the problem is that the definition of "happification" I used
resulted in not diverse enough results. I wanted something which would give me
numbers where any of the digits could be anything. Something more random.

I considered using a hash instead, like md5, but that has its own problems.
There's no gaurantee that any number would actually reach `000001`, which isn't
required but it's a nice feature that I wanted. It also would be unlikely that
there would be any melancoils that weren't absolutely gigantic.

I ended up redefining what it meant to happify a hexadecimal number. Instead of
adding all the digits up, I first split up the red, green, and blue digits into
their own numbers, happified those numbers, and finally reassembled the results
back into a single number. For example:

```
FF5500
FF, 55, 00
F*F + F*F, 5*5 + 5*5, 0*0 + 0*0
1C2, 32, 00
C23200
```

I drop that 1 on the `1C2`, because it has no place in this system. Sorry 1.

Simply replacing that function resulted in this image (click for 5000x5000) version:

[![Result of second attempt](/img/happy-tree-colors-small.png)](/img/happy-tree-colors.png)

The first thing you notice is that it's so colorful! So that goal was achieved.

The second thing you notice is that there's *significantly* more melancoils.
Hundreds, even. I'll list out a couple of them here:

```
00000D 0000A9 0000B5 000092 000055 000032
000D0D 00A9A9 00B5B5 009292 005555 003232
0D0D0D A9A9A9 B5B5B5 929292 555555 323232
0D0D32 A9A90D B5B5A9 9292B5 555592 323255
...
```

And so on. Those numbers are actually the same numbers from the
melancoil in the first attempt. This make sense, since the happification
algorithm didn't really change, just how we applied it, so the melancoil numbers
didn't change, just where they appear. And they appear *a lot*. Every
conceivable combination of those three two-digit places across 6 numbers.

There's not really much we can do about this, it's simply a property of the
algorithm chosen. Either image is valid, it's up to the reader to decide which
they like more!
