#!/usr/bin/env python3
"""
Mock script to simulate rtl_433 JSON output for Ecowitt WS90 weather station.
Generates realistic weather data with independent random patterns for each metric.
"""

import json
import random
import sys
import time
from datetime import datetime, timezone, timedelta

def generate_weather_data(start_date, end_date):
    """Generate realistic Mediterranean weather data for a given date range"""
    data = []
    current_date = start_date
    
    while current_date <= end_date:
        # Get month and day for seasonal variations
        month = current_date.month
        day = current_date.day
        hour = current_date.hour
        
        # Mediterranean climate characteristics:
        # - Hot, dry summers (June-September)
        # - Mild, wet winters (December-February)
        # - Transitional spring (March-May) and autumn (October-November)
        
        # Temperature: Mediterranean seasonal pattern
        if month in [6, 7, 8]:  # Summer: hot and dry
            base_temp = 28 + random.gauss(0, 3)
            # Diurnal variation: cooler at night
            if hour < 6 or hour > 20:
                base_temp -= 8
        elif month in [12, 1, 2]:  # Winter: mild and wet
            base_temp = 12 + random.gauss(0, 3)
        elif month in [3, 4, 5]:  # Spring: warming up
            base_temp = 16 + (month - 3) * 2 + random.gauss(0, 3)
        else:  # Autumn: cooling down
            base_temp = 24 - (month - 9) * 2 + random.gauss(0, 3)
        
        temp = base_temp + random.gauss(0, 2)
        temp = max(0, min(45, temp))  # Clamp between 0°C and 45°C
        
        # Humidity: Inversely related to temperature
        # Summer: 40-60%, Winter: 70-85%
        if month in [6, 7, 8]:  # Summer: dry
            base_humidity = 50
        elif month in [12, 1, 2]:  # Winter: humid
            base_humidity = 75
        else:
            base_humidity = 60
        
        humidity = base_humidity + random.gauss(0, 10)
        humidity = max(30, min(95, humidity))
        
        # UV Index: Very high in summer, low in winter
        if month in [6, 7, 8]:  # Summer: very high UV
            if 10 <= hour <= 16:  # Peak daylight hours
                uv_index = random.randint(8, 11)
            elif 6 <= hour < 10 or 16 < hour <= 19:  # Morning/evening
                uv_index = random.randint(4, 7)
            else:  # Night
                uv_index = 0
        elif month in [12, 1, 2]:  # Winter: low UV
            if 10 <= hour <= 15:
                uv_index = random.randint(1, 3)
            else:
                uv_index = 0
        else:  # Spring/Autumn: moderate UV
            if 10 <= hour <= 16:
                uv_index = random.randint(5, 8)
            elif 6 <= hour < 10 or 16 < hour <= 19:
                uv_index = random.randint(2, 5)
            else:
                uv_index = 0
        
        # Light: High in summer, lower in winter
        if month in [6, 7, 8]:  # Summer: bright
            if 6 <= hour <= 20:  # Daytime
                base_light = 80000 + random.gauss(0, 15000)
            else:  # Night
                base_light = 0
        elif month in [12, 1, 2]:  # Winter: lower light
            if 8 <= hour <= 17:
                base_light = 40000 + random.gauss(0, 10000)
            else:
                base_light = 0
        else:  # Spring/Autumn
            if 7 <= hour <= 19:
                base_light = 60000 + random.gauss(0, 12000)
            else:
                base_light = 0
        
        light_lux = int(max(0, min(120000, base_light)))
        
        # Wind: Moderate, higher in winter
        if month in [12, 1, 2]:  # Winter: windier
            base_wind = 6 + random.gauss(0, 2)
        elif month in [6, 7, 8]:  # Summer: calmer
            base_wind = 3 + random.gauss(0, 1.5)
        else:
            base_wind = 4.5 + random.gauss(0, 2)
        
        wind_speed = max(0, base_wind + random.gauss(0, 2))
        wind_speed = min(25, wind_speed)
        
        # Wind gust: 1.5-2x wind speed
        wind_gust = wind_speed * (1.5 + random.random() * 0.5)
        wind_gust = max(0, min(40, wind_gust))
        
        # Wind direction: Predominantly from NW in summer, SW in winter
        if month in [6, 7, 8]:  # Summer: NW winds
            base_dir = 315
            wind_dir = int(base_dir + random.gauss(0, 30)) % 360
        elif month in [12, 1, 2]:  # Winter: SW winds
            base_dir = 225
            wind_dir = int(base_dir + random.gauss(0, 30)) % 360
        else:  # Transitional
            wind_dir = random.randint(0, 359)
        
        # Rainfall: Mediterranean pattern
        # - Dry summers (rare rain)
        # - Wet winters (frequent rain)
        # Note: Values are scaled for 5-second intervals (divide hourly rates by 720)
        if month in [6, 7, 8]:  # Summer: very dry
            rain_probability = 0.05  # 5% chance of rain
            rain_intensity = random.uniform(0.0007, 0.007)  # Light showers (0.5-5 mm/hour / 720)
        elif month in [12, 1, 2]:  # Winter: wet
            rain_probability = 0.30  # 30% chance of rain
            rain_intensity = random.uniform(0.0028, 0.021)  # Moderate to heavy (2-15 mm/hour / 720)
        elif month in [3, 4, 5]:  # Spring: moderate
            rain_probability = 0.15
            rain_intensity = random.uniform(0.0014, 0.014)  # Moderate (1-10 mm/hour / 720)
        else:  # Autumn: moderate
            rain_probability = 0.20
            rain_intensity = random.uniform(0.0014, 0.017)  # Moderate (1-12 mm/hour / 720)
        
        if random.random() < rain_probability:
            rain_mm = round(rain_intensity, 1)
        else:
            rain_mm = 0
        
        # Battery: 95% chance of being ok
        battery_ok = random.choice(['ok', 'low']) if random.random() < 0.05 else 'ok'
        
        # Create the weather data point
        # Note: wind_speed_m_s and wind_gust_m_s are in meters per second (convert from km/h)
        data_point = {
            "time": current_date.strftime("%Y-%m-%dT%H:%M:%SZ"),
            "model": "Ecowitt-WH90",
            "id": 1234,
            "temperature_C": round(temp, 1),
            "humidity": int(humidity),
            "wind_speed_m_s": round(wind_speed / 3.6, 1),  # Convert km/h to m/s
            "wind_gust_m_s": round(wind_gust / 3.6, 1),    # Convert km/h to m/s
            "wind_dir_deg": wind_dir,
            "rain_mm": rain_mm,
            "uv": uv_index,
            "light_lux": light_lux,
            "battery": battery_ok
        }
        
        data.append(data_point)
        
        # Move to next minute
        current_date += timedelta(minutes=1)
    
    return data

def main():
    """Main loop to continuously generate weather data."""
    try:
        while True:
            # Generate a single reading for the current time
            weather_data = generate_weather_data(datetime.now(), datetime.now())
            # Output the single reading
            if weather_data:
                print(json.dumps(weather_data[0]))
                sys.stdout.flush()
            # Wait 5 seconds before next reading
            time.sleep(5)
    except KeyboardInterrupt:
        # Exit gracefully on Ctrl+C
        pass

if __name__ == "__main__":
    main()