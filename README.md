# linear-future

Future-dated tickets in Linear.

For every future-dated ticket:
- Add label "Later"
- Add @YYYY-MM-DD prefix to the title

Run this tool every day, it will remove "Later" labels from any ticket that is no longer in the future.

It also creates issues from templates based on recurrence tags in the template name. Supported tags:
- `@Daily`
- `@Mon @Tue` or `@Mon,Tue` (weekly)
- `@1` `@3` `@-1` (first/third/last day of every month)
- `@Jan1` `@Jan3` `@Jan-1` (first/third/last day of every January)
- `@Jan,Jun1` (first day of every January and June)
