Version v0.3.0
==============

* MISC: Changes required for v0.3.0 of github.com/tbogdala/fizzle

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
