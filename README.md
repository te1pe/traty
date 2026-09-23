# Traty

A personal expense tracker: record a spend, see what went out today and since
Monday, break it down by category, build a report for any period and export it
to CSV.

Go + SQLite + server-rendered HTML. No Docker, no external database, no
JavaScript framework. Russian and English interface, light / dark / system
themes, mobile and desktop layouts.

## Running it

```bash
go run .
```

The app comes up on http://localhost:8080. On first run it creates the database
file `data/finance.db` along with the tables and indexes — nothing to set up by
hand.

At startup it prints two addresses (the startup log is in Russian):

```
Траты: база data/finance.db
  на этом компьютере: http://localhost:8080
  с телефона в той же сети Wi-Fi: http://192.168.3.10:8080
```

Building a binary:

```bash
go build -o traty .
./traty
```

## Opening it from a phone

1. The computer and the phone must be on the same Wi-Fi network.
2. Run `go run .` and look for the `http://192.168.x.x:8080` address in the output.
3. Open that address in Safari or Chrome on the phone.
4. To make it open like a regular app, with an icon and no address bar: in
   Safari use Share → "Add to Home Screen", in Chrome use menu → "Add to
   Home screen".

If the page does not load, check that macOS is not blocking incoming
connections: System Settings → Network → Firewall. While the computer is
asleep or the app is stopped, the phone will not open the page — the data lives
on the computer, this is not a cloud service.

## Tests

```bash
go test ./...
```

Linting:

```bash
gofmt -l .
go vet ./...
```

## Configuration

The defaults are sensible; environment variables are rarely needed.

| Variable | Default | What it sets |
|---|---|---|
| `PORT` | `8080` | HTTP server port |
| `DATABASE_PATH` | `data/finance.db` | path to the SQLite file |

```bash
PORT=3000 DATABASE_PATH=/var/lib/traty/finance.db go run .
```

Currency, language and theme are changed on the Settings page and stored in the
database.

## Project layout

```
main.go                  startup, configuration, routes, flash messages
store.go                 SQLite: schema, models, every SQL query
money.go                 money in minor units: parsing and formatting
dates.go                 local dates, weeks starting Monday, date formats
i18n.go                  interface strings in ru/en, plural forms
chart.go                 donut chart computed on the server
form.go                  the expense form and its validation
view.go                  templates and their helpers
handlers_overview.go     dashboard and the category sheet
handlers_expenses.go     creating, editing and deleting an expense
handlers_reports.go      reports for a period
handlers_categories.go   categories
handlers_settings.go     settings
csv.go                   CSV export
templates/               HTML templates
static/css/              tokens.css and bundle.css from the design system, app.css — overrides
static/icons/            icons for the home screen
static/manifest.webmanifest  manifest so the app installs on a phone
data/                    the database file (created on startup)
```

A single `main` package with no extra layers: for an app this size that is the
clearest structure. All the SQL lives in `store.go`, so swapping the storage or
adding authentication later does not touch anything else.

## What it does

**Overview.** Today's total in large type, a "This week" card (Monday → today,
not the last 7 days) with a donut chart by category, and the five most recent
expenses. Clicking a segment or a legend row opens the category detail with its
total, the number of expenses and the list; the period is a week or a month.

**Expenses.** Three taps to add one: "+" → amount → category → "Save". The
amount accepts both a comma and a dot, the date defaults to today (there is a
"Yesterday" shortcut and a date picker), the note is optional. An expense can
be opened, edited and deleted with confirmation. A new category can be created
right inside the expense form.

**Categories.** Created by the user; nothing is seeded. Names are unique
case-insensitively, Cyrillic included. A colour from the design system palette
is assigned automatically. A category that has expenses cannot be deleted — the
app explains why.

**Reports.** Preset periods (week, month, last month, year) or an arbitrary
date range. A report shows the total, the number of expenses, a breakdown by
category with shares, and the expenses grouped by day.

**CSV.** `Date;Category;Amount;Currency;Note` (the header follows the interface
language), UTF-8 with a BOM so Excel opens it correctly, `;` as the separator
and a comma as the decimal mark. Values starting with `=`, `+`, `-` or `@` are
escaped — protection against CSV injection.

**Settings.** Currency (EUR, USD, RUB, GBP), language (Russian, English) and
theme (light, dark, system). Saved immediately on selection.
