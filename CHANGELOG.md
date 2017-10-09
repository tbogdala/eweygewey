Since Version v0.3.1
====================

* New: Added Manager.NewFontBytes() to load a font by byte slice so that
  clients can load embedded fonts.

Version v0.3.1
==============

* BUG: Fixed issue #3 where the VBO data was getting corrupted by attempting
  to add zero faces.

Version v0.3.0
==============

* MISC: Changes required for v0.3.0 of github.com/tbogdala/fizzle inclusing using
  the new Material type and the new built-in shaders.

Version v0.2.0
==============

* BUG: Fixed Manager.RemoveWindow() bug with indexing a slice incorrectly.
* BUG: Fixed editboxes with too long of text overflowing the widget.

* NEW: Manager.GetWindowsByFilter() to get UI Windows using a function
  to filter the list.

* NEW: Font.CreateTextAdv() for advanced control of text creation -- useful for
  the editbox widget -- to create text of a maximum width starting at a custom
  spot in the string.

* NEW: Font.OffsetForIndexAdv() for advance control while getting the offset
  in pixels for a location in a string based on a custom starting spot in the string.

* MISC: Added an editbox with too long of a string to display at once in
  the main example application.
