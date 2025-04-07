package opera

import "encoding/json"

func UpdateRules(src Rules, diff []byte) (Rules, error) {
	changed := src.Copy()
	if err := json.Unmarshal(diff, &changed); err != nil {
		return Rules{}, err
	}

	// protect readonly fields
	changed.NetworkID = src.NetworkID
	changed.Name = src.Name

	// check validity of the new rules
	if changed.Upgrades.Allegro {
		if err := changed.Validate(); err != nil {
			return Rules{}, err
		}
	}
	return changed, nil
}
