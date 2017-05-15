EweyGewey v0.3.1
================

EweyGewey is an OpenGL immediate-mode GUI library written in the [Go][golang] programming 
language that is designed to be integrated easily into an OpenGL application.

The design of the library is heavily inspired by [imgui][imgui].

UNDER CONSTRUCTION
==================

At present, it is very much in an alpha stage with new development adding in
features, widgets and possibly API breaks. Any API break should increment the
minor version number and any patch release tags should remain compatible even
in development 0.x versions.

Screenshots
-----------

Here's some of what's available right now in the [basic example][basic_example]:

![basic_ss][basic_ss]


Requirements
------------

* [Mathgl][mgl] - for 3d math
* [Freetype][ftgo] - for dynamic font texture generation
* [Fizzle][fizzle] - provides an OpenGL 3/es2/es3 abstraction
* [GLFW][glfw-go] (v3.1) - currently GLFW is the only 'host' support for input

Additionally, a backend graphics provider needs to be used. At present, fizzle
supports the following:

* [Go GL][go-gl] - pre-generated OpenGL bindings using their glow project
* [opengles2][opengles2] - Go bindings to the OpenGL ES 2.0 library

These are included when the `graphicsprovider` subpackage is used and direct
importing is not required.

Installation
------------

The dependency Go libraries can be installed with the following commands.

```bash
go get github.com/go-gl/glfw/v3.1/glfw
go get github.com/go-gl/mathgl/mgl32
go get github.com/golang/freetype
go get github.com/tbogdala/fizzle
```

An OpenGL library will also be required for desktop applications; install
the OpenGL 3.3 library with the following command:

```bash
go get github.com/go-gl/gl/v3.3-core/gl
```

If you're compiling for Android/iOS, then you will need an OpenGL ES library,
and that can be installed with the following command instead:

```bash
go get github.com/remogatto/opengles2
```

This does assume that you have the native GLFW 3.1 library installed already
accessible to Go tools. This should be the only native library needed.

Current Features
----------------

* Basic windowing system
* Basic theming support
* Basic input support that detects mouse clicks and double-clicks
* Basic scaling for larger resolutions
* Widgets:
    * Text
    * Buttons
    * Sliders for integers and floats with ranges and without
    * Scroll bars
    * Images
    * Editbox
    * Checkbox
    * Separator
    * Custom drawn 3d widgets

TODO
----

The following need to be addressed in order to start releases:

* more widgets:
    * text wrapping
    * multi-line text editors
    * combobox
    * image buttons
* detailed theming (e.g. custom drawing of slider cursor)
* texture atlas creation
* z-ordering for windows
* scroll bars don't scroll on mouse drag
* editbox cursor doesn't start where mouse was clicked
* text overflow on editboxes isn't handled well
* better OpenGL flag management
* documentation
* samples


LICENSE
=======

EweyGewey is released under the BSD license. See the [LICENSE][license-link] file for more details.

Fonts in the `examples/assets` directory are licensed under the [SIL OFL][sil_ofl] open font license.

[golang]: https://golang.org/
[fizzle]: https://github.com/tbogdala/fizzle
[glfw-go]: https://github.com/go-gl/glfw
[mgl]: https://github.com/go-gl/mathgl
[ftgo]: https://github.com/golang/freetype
[go-gl]: https://github.com/go-gl/glow
[opengles2]: https://github.com/remogatto/opengles2
[imgui]: https://github.com/ocornut/imgui
[sil_ofl]: http://scripts.sil.org/cms/scripts/page.php?site_id=nrsi&id=OFL
[license-link]: https://raw.githubusercontent.com/tbogdala/eweygewey/master/LICENSE
[basic_ss]: https://github.com/tbogdala/eweygewey/blob/master/examples/screenshots/basic_ss_0.jpg
[basic_example]: https://github.com/tbogdala/eweygewey/blob/master/examples/basicGLFW/main.go
