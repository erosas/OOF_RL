'use strict';

// --- Arena name mapping ---
// Keys are the internal RL/BC map codes (lowercased). Source: https://ballchasing.com/api/maps
const ARENA_NAMES = {
  // Champions Field
  cs_p:                    'Champions Field',
  cs_day_p:                'Champions Field (Day)',
  cs_hw_p:                 'Rivals Arena',
  swoosh_p:                'Champions Field (Nike FC)',
  bb_p:                    'Champions Field (NFL)',

  // Mannfield
  eurostadium_p:           'Mannfield',
  eurostadium_night_p:     'Mannfield (Night)',
  eurostadium_rainy_p:     'Mannfield (Stormy)',
  eurostadium_dusk_p:      'Mannfield (Dusk)',
  eurostadium_snownight_p: 'Mannfield (Snowy)',

  // Starbase ARC
  arc_p:                   'Starbase ARC',
  arc_standard_p:          'Starbase ARC (Standard)',
  arc_darc_p:              'Starbase ARC (Aftermath)',

  // DFH Stadium
  stadium_p:               'DFH Stadium',
  stadium_day_p:           'DFH Stadium (Day)',
  stadium_winter_p:        'DFH Stadium (Snowy)',
  stadium_foggy_p:         'DFH Stadium (Stormy)',
  stadium_race_day_p:      'DFH Stadium (Circuit)',

  // Urban Central
  trainstation_p:          'Urban Central',
  trainstation_dawn_p:     'Urban Central (Dawn)',
  trainstation_night_p:    'Urban Central (Night)',
  trainstation_spooky_p:   'Urban Central (Spooky)',
  haunted_trainstation_p:  'Urban Central (Haunted)',

  // Beckwith Park
  park_p:                  'Beckwith Park',
  park_night_p:            'Beckwith Park (Midnight)',
  park_bman_p:             'Beckwith Park (Night)',
  park_rainy_p:            'Beckwith Park (Stormy)',
  park_snowy_p:            'Beckwith Park (Snowy)',

  // Wasteland
  wasteland_p:             'Wasteland',
  wasteland_s_p:           'Wasteland (Standard)',
  wasteland_night_p:       'Wasteland (Night)',
  wasteland_night_s_p:     'Wasteland (Standard, Night)',
  wasteland_grs_p:         'Wasteland (Pitched)',

  // Neo Tokyo
  neotokyo_p:              'Neo Tokyo',
  neotokyo_standard_p:     'Neo Tokyo (Standard)',
  neotokyo_arcade_p:       'Neo Tokyo (Arcade)',
  neotokyo_hax_p:          'Neo Tokyo (Hacked)',
  neotokyo_toon_p:         'Neo Tokyo (Comic)',

  // Utopia Coliseum
  utopiastadium_p:         'Utopia Coliseum',
  utopiastadium_dusk_p:    'Utopia Coliseum (Dusk)',
  utopiastadium_snow_p:    'Utopia Coliseum (Snowy)',
  utopiastadium_lux_p:     'Utopia Coliseum (Gilded)',

  // AquaDome
  underwater_p:            'Aquadome',
  underwater_grs_p:        'AquaDome (Salty Shallows)',

  // Salty Shores
  beach_p:                 'Salty Shores',
  beach_night_p:           'Salty Shores (Night)',
  beach_night_grs_p:       'Salty Shores (Salty Fest)',
  beachvolley:             'Salty Shores (Volley)',

  // Forbidden Temple
  chn_stadium_p:           'Forbidden Temple',
  chn_stadium_day_p:       'Forbidden Temple (Day)',
  fni_stadium_p:           'Forbidden Temple (Fire & Ice)',

  // Farmstead
  farm_p:                  'Farmstead',
  farm_night_p:            'Farmstead (Night)',
  farm_grs_p:              'Farmstead (Pitched)',
  farm_hw_p:               'Farmstead (Spooky)',
  farm_upsidedown_p:       'Farmstead (The Upside Down)',

  // Throwback Stadium
  throwbackstadium_p:      'Throwback Stadium',
  throwbackhockey_p:       'Throwback Stadium (Snowy)',

  // Neon Fields
  music_p:                 'Neon Fields',

  // Dunk House / Hoops
  hoopsstadium_p:          'Dunk House',
  hoopsstreet_p:           'The Block (Dusk)',

  // Deadeye Canyon
  outlaw_p:                'Deadeye Canyon',
  outlaw_oasis_p:          'Deadeye Canyon (Oasis)',

  // Knockout arenas
  ko_calavera_p:           'Calavera',
  ko_carbon_p:             'Carbon',
  ko_quadron_p:            'Quadron',

  // Labs
  labs_basin_p:            'Basin',
  labs_circlepillars_p:    'Pillars',
  labs_corridor_p:         'Corridor',
  labs_cosmic_p:           'Cosmic',
  labs_cosmic_v4_p:        'Cosmic',
  labs_doublegoal_p:       'Double Goal',
  labs_doublegoal_v2_p:    'Double Goal',
  labs_galleon_p:          'Galleon',
  labs_galleon_mast_p:     'Galleon Retro',
  labs_holyfield_p:        'Loophole',
  labs_octagon_p:          'Octagon',
  labs_octagon_02_p:       'Octagon',
  labs_pillarglass_p:      'Hourglass',
  labs_pillarheat_p:       'Barricade',
  labs_pillarwings_p:      'Colossus',
  labs_underpass_p:        'Underpass',
  labs_underpass_v0_p:     'Underpass',
  labs_utopia_p:           'Utopia Retro',

  // Other
  shattershot_p:           'Core 707',
  ff_dusk_p:               'Estadio Vida (Dusk)',
  street_p:                'Sovereign Heights (Dusk)',
  woods_p:                 'Drift Woods',
  woods_night_p:           'Drift Woods (Night)',
};

function friendlyArena(code) {
  if (!code) return '—';
  const key = code.toLowerCase().trim();
  return ARENA_NAMES[key] ||
    key.replace(/_p$/, '').replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase());
}

const PLAYLIST_NAMES = {
  1:  'Casual 1v1',  2:  'Casual 2v2',  3:  'Casual 3v3',
  4:  'Custom',      6:  'Private',      10: 'Ranked 1v1',
  11: 'Ranked 2v2',  13: 'Ranked 3v3',  14: 'Solo 3v3',
  22: 'Tournament',  27: 'Rocket Labs',  28: 'Rumble',
  29: 'Dropshot',    30: 'Hoops',        31: 'Snow Day',
  34: 'Casual Chaos',35: 'Gridiron',     41: 'Heatseeker',
  43: 'Spike Rush',
};

function friendlyPlaylist(id) {
  if (id == null) return '';
  return PLAYLIST_NAMES[id] || '';
}