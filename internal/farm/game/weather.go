package game

import (
	"context"

	"github.com/it00021hot/qq-farm-core/internal/farm/proto/weatherpb"
)

// GetWeatherStatus fetches the current farm's weather (WeatherService).
func (a *API) GetWeatherStatus(ctx context.Context) (*weatherpb.GetWeatherStatusReply, error) {
	raw, _, err := a.Sender.Send(ctx, "gamepb.weatherpb.WeatherService", "GetWeatherStatus", nonNilBody(nil))
	if err != nil {
		return nil, err
	}
	reply := &weatherpb.GetWeatherStatusReply{}
	if err := unmarshalMessage(raw, reply); err != nil {
		return nil, err
	}
	return reply, nil
}
