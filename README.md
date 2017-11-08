# databox-driver-fitbit

Databox driver for the Fitbit API

By Chris Greenhalgh <chris.greenhalgh@nottingham.ac.uk>,
Copyright (C) The University of Nottingham, 2017

Status: syncs profile (only)

Roadmap:
- implement initial day summary activity support
- more datasources (sleep, weight, intra-day activity, heartrate, intra-day heartrate)
- support oauth from databox app (requires changes to app)

## Data sources

#### Profile

This driver downloads your profile from fitbit into a key-value in Databox. 
- datasource type: `Fitbit-Profile`
- store type: `store-json`
- API: kv
- content type: `application/json`
- Schema: see below

Profile is a JSON object with fields:
- `displayName`
- `fullName`
- `offsetFromUTCMillis`
- `timezone`

This is a strict subset of the information in the "user" object from [get profile](https://dev.fitbit.com/reference/web-api/user/#get-profile).

### Devices

This downloads device info from fitbit into a key-value in Databox.
- datasource type: `Fitbit-Devices`
- store type: `store-json`
- API: kv
- content type: `application/json`
- Schema: see below

Profile is a JSON array containing a JSON object per user device with fields:
- `battery`, e.g. "High"
- `deviceVersion`, e.g. "Charge HR"
- `id`, e.g. "12345"
- `lastSyncTime`, e.g. "2015-07-27T17:01:39.313"
- `type`, e.g. "TRACKER", "SCALE"

This is the information from [get devices](https://dev.fitbit.com/reference/web-api/devices/#get-devices).

### Daily Activity Summary

(to do)

This driver downloads your daily activity summary from fitbit into a time-series in Databox. 
- datasource type: `Fitbit-Activity-DaySummary`
- store type: `store-json`
- API: time-series
- content type: `application/json`
- Schema: see below

Each activity is a JSON object with fields:
- `date`: in format `yyyy-MM-dd` (not present in Fitbit response but added in driver)
- `timezone`: (not present in Fitbit response; added from GetProfile response)
- `activityCalories`: (int) 
- `caloriesBMR`: (int)
- `distances`: array of {activity,distance} sub-objects (activities: "tracker", "total", ...; distance km float)
- `fairlyActiveMinutes`: (int)
- `lightlyActiveMinutes`: (int)
- `sedentaryMinutes`: (int)
- `steps`: (int)
- `veryActiveMinutes`: (int)

This is (except for date and timezone) a strict subset of the information in the [Fitbit Daily summary](https://dev.fitbit.com/reference/web-api/activity/#get-daily-activity-summary).

The store timestamp is the day's date, T12:00:00 (i.e. midday), in their preferred timezone.

## Install / use

To install from local build:
```
./databox-install-component cgreenhalgh/driver-fitbit
```

### Configure

Note: currently authorizing the driver will only work from a browser on the machine running the databox, not from the databox app or from a remote browser.

1. In the Databox UI install the driver, called "Fitbit". 
1. Open the driver UI and "Link to Fitbit account"; log into Strava if required and then authorize access.
1. (Probably best for now to re-open the databox UI, or you can switch to the app)
1. In the driver UI "Sync data from Fitbit"; wait a few seconds and reload the page to see the updated status (at some point it will be made to auto-update)

See the [activity summary app](https://github.com/cgreenhalgh/databox-app-activity-summary) for a possible way to view your activity data (in the future).

### Using a personal Fitbit app

Note that the driver by default authenticates with a test/demo fitbit
app (NB not databox app). This app does NOT have access to a user's
intra-day activity or heartrate data. But if you create your own 
"Personal" fitbit app and link to that then that will (potentially)
give access to intra-day data.

You can link the driver to your own fitbit app by:

1. Creating a personal [fitbit app](https://dev.fitbit.com/apps/new). For the authorization callback domain put 'http://localhost:8989/driver-fitbit/ui/hash_auth_callback'.
1. In the driver UI replacing the `client_id` (?and `client_secret`) with the values for your new app and hitting "Configure"
1. In the driver UI "Link to Strava account" again.

## Implementation notes

see [implementation notes](docs/implementation-notes.md)
