# Run Plan Generator

Allow users to use canned training plans for upcoming races or create their own.

Now implemented:

- Repeated segments in yaml templates (e.g 6x [100 yards zone 5, recover 2 mins])
- Allow export to Google calendar etc.
- Allow user to calculate their heart rate zones using either max heart rate,
  heart rate reserve, or lactate threshold heart rate, if known.
- Template runs are still having some trouble with offsets from the correct days
of the week.

TODO:

- Allow users to construct multi-segment runs with repeats etc.
- Allow zone bpm refreshes mid-plan if a user's LTHR or resting HR changes.
- Makefile and other installation instructions.
