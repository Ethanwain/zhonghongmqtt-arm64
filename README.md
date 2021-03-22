# Zhonghong MQTT

An integration for [Zhonghong thermostat gateway](http://zhonghongtech.cn/v1/product.shtml) and MQTT brokers, in order to control thermostats with [Home Assistant](https://www.home-assistant.io/), running with Docker.

## Overview

The gateway supports many interfaces and protocols, such as Zigbee, TCP, HTTP and RS485.

For efficiency, we use the HTTP API temporarily.

## Installing

### Download Docker image

```
docker pull halozheng/zhonghong-mqtt
```

### Create the container

* Create the config file with a name ```config.yml```
```
Gateway:
  Host: 'Your gateway IP address'
  Port: 80
  Username: 'admin'
  Password: ''
MQTT:
  Host: 'Your MQTT Broker IP address'
  Port: 1883
  Username: 'Your MQTT username'
  Password: 'Your MQTT password'
```
* Create the container with volume mapping ```/your-path/config.yml => /config.yml```
* Run the container

## Getting Started

```
x: Outside machine sequence number, in most cases it is 1
y: Inside machine sequence number, from 1 to N
```

## State Topics

```
zhonghong/x/y/mode/state
Mode state of the thermostat, available values: heat|dry|cool|fan_only|off.

zhonghong/x/y/temperature/state
Set temperature of the thermostat, available values: number with celsius.

zhonghong/x/y/fan/state
Fan speed of the thermostat, available values: low|medium|high.

zhonghong/x/y/current_temperature/state
Current temperature with celsius of the thermostat.

```

## Command Topics


```
zhonghong/x/y/mode/set
zhonghong/x/y/temperature/set
zhonghong/x/y/fan/set
```

## Integration with Home Assistant

```
climate:
  - platform: mqtt
    name: 'My Thermostat'
    modes:
      - 'heat'
      - 'dry'
      - 'cool'
      - 'fan_only'
      - 'off'
    fan_modes:
      - 'low'
      - 'medium'
      - 'high'
    max_temp: 30
    min_temp: 18
    mode_command_topic: 'zhonghong/1/1/mode/set'
    mode_state_topic: 'zhonghong/1/1/mode/state'
    temperature_command_topic: 'zhonghong/1/1/temperature/set'
    temperature_state_topic: 'zhonghong/1/1/temperature/state'
    fan_mode_command_topic: 'zhonghong/1/1/fan/set'
    fan_mode_state_topic: 'zhonghong/1/1/fan/state'
    current_temperature_topic: 'zhonghong/1/1/current_temperature/state'
```

## FAQ

### My gateway does not support Wi-Fi

We can use a normal wireless router, run with client mode, to convert the Wi-Fi to RJ45 interface.

Such as [TP-LINK TL-WR800N](https://item.jd.com/524759.html).
