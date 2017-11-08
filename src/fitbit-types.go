package main

// Fitbit types - as saved in store(s)

// Fitbit profile information
type FitbitProfile struct{
	DisplayName string `json:"displayName"`
	FullName string `json:"fullName"`
	OffsetFromUTCMillis int `json:"offsetFromUTCMillis"`
	Timezone string `json:"timezone"`
}

type FitbitActivityDistance struct {
	Activity string `json:"activity"` //"tracker", "total", ...
	Distance float64 `json:"distance"` //km float
}

// Fitbit daily activity summary 
type FitbitDaySummary struct {
	Date string `json:"date"` // in format `yyyy-MM-dd` (not present in Fitbit response but added in driver)
	Timezone string `json:"timezone"` // (not present in Fitbit response; added from GetProfile response)
	OffsetFromUTCMillis int `json:"offsetFromUTCMillis"` // (not present in Fitbit response; added from GetProfile response)
	ActivityCalories int `json:"activityCalories"` 
	CaloriesBMR int `json:"caloriesBMR"`
	Distances []FitbitActivityDistance `json:"distances"`
	FairlyActiveMinutes int `json:"fairlyActiveMinutes"`
	LightlyActiveMinutes int `json:"lightlyActiveMinutes"`
	SedentaryMinutes int `json:"sedentaryMinutes"`
	Steps int `json"steps"`
	VeryActiveMinutes int `json:"veryActiveMinutes"`
}

type FitbitDaySummaryDSE struct {
	Data FitbitDaySummary `json:"data"`
	Timestamp int64 `json:"timestamp"`
}