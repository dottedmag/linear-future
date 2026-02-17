# linear-future

Creates issues from Linear templates based on schedule lines in the template description.

For one-time issues, use `At:` with a specific date:

```
At: 2025-06-15
```

For recurring issues, use `Recurrence:` with a pattern:

```
Recurrence: daily
Recurrence: Mon
Recurrence: Tue
Recurrence: 1
Recurrence: 15
Recurrence: last
Recurrence: Jan 1
Recurrence: Jun last
```

Each recurrence value is one of:
- `daily` — every day
- A three-letter weekday (`Mon`, `Tue`, etc.) — that day every week
- A number `1`–`31` — that day every month
- `last` — last day of every month
- A three-letter month + number (`Jan 1`) — that day in that month
- A three-letter month + `last` (`Jun last`) — last day of that month

Multiple lines (of any kind) are OR'd — any match triggers issue creation.
