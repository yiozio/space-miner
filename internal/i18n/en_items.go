package i18n

import "github.com/yiozio/space-miner/internal/entity"

// newENItems は英語のアイテム文字列セットを返す。
func newENItems() ItemsStrings {
	return ItemsStrings{
		Parts: map[int]ItemText{
			int(entity.PartIDCockpit):         {Name: "Cockpit", Desc: "Pilot seat. Required. Provides minimal thrust if no thrusters are installed."},
			int(entity.PartIDGunStarter):      {Name: "Starter Gun", Desc: "Factory-issue popgun. Low damage, slow rate."},
			int(entity.PartIDGunMkI):          {Name: "Gun Mk-I", Desc: "Standard forward gun."},
			int(entity.PartIDGunMkII):         {Name: "Gun Mk-II", Desc: "Heavy gun. High damage, slow rate."},
			int(entity.PartIDGunRapid):        {Name: "Rapid Gun", Desc: "Light gun. Fast rate, low damage."},
			int(entity.PartIDGunPlasma):       {Name: "Plasma Cannon", Desc: "Slow plasma orbs. High damage with impact burst."},
			int(entity.PartIDGunLaser):        {Name: "Laser Pulse", Desc: "Pinpoint laser pulses. Very fast, light damage."},
			int(entity.PartIDGunRocket):       {Name: "Rocket Launcher", Desc: "Slow heavy rounds. Deals splash damage on impact."},
			int(entity.PartIDThrusterStarter): {Name: "Starter Thruster", Desc: "Factory-issue cheap engine. Minimal thrust."},
			int(entity.PartIDThrusterStd):     {Name: "Thruster", Desc: "Standard engine."},
			int(entity.PartIDThrusterLight):   {Name: "Light Thruster", Desc: "Compact engine. Lower thrust, fuel-efficient."},
			int(entity.PartIDThrusterHeavy):   {Name: "Heavy Thruster", Desc: "High-output engine. Strong thrust, hungry."},
			int(entity.PartIDFuelStd):         {Name: "Fuel Tank", Desc: "Standard fuel tank."},
			int(entity.PartIDCargoStd):        {Name: "Cargo", Desc: "Resource storage. Increases cargo capacity."},
			int(entity.PartIDArmorStd):        {Name: "Armor", Desc: "Hardened plating. Increases max HP."},
			int(entity.PartIDShieldStd):       {Name: "Shield", Desc: "Shield generator. Absorbs damage; regenerates after 2s without damage."},
			int(entity.PartIDAutoAimStd):      {Name: "Auto-Aim", Desc: "Beams the last-hit asteroid grid by grid. Damage over time."},
			int(entity.PartIDWarpStd):         {Name: "Warp", Desc: "Warp drive."},
			int(entity.PartIDMineLayer):       {Name: "Mine Layer", Desc: "Deploys a mine on fire. The mine bursts into 6 bullets after ~1s."},
			int(entity.PartIDDroneStd):        {Name: "Attack Drone", Desc: "Deploys an autonomous drone on fire. For ~10s it beams the nearest asteroid or enemy."},
			int(entity.PartIDDroneGun):        {Name: "Gun Drone", Desc: "Deploys an autonomous drone on fire. For ~10s it shoots bullets at the nearest asteroid or enemy."},
		},
		Resources: map[int]ItemText{
			int(entity.ResourceIron):   {Name: "IRON"},
			int(entity.ResourceBronze): {Name: "BRONZE"},
			int(entity.ResourceIce):    {Name: "ICE"},
		},
		PiratePatterns: map[int]ItemText{
			int(entity.PiratePatternScout):   {Name: "Scout"},
			int(entity.PiratePatternBrawler): {Name: "Brawler"},
			int(entity.PiratePatternCruiser): {Name: "Cruiser"},
		},
	}
}
