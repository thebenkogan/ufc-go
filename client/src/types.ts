export type Event = {
  id: string;
  start_time: string;
  fights: Fight[];
};

export type Fight = {
  fighters: string[];
  winner?: string;
};

export type Picks = {
  winners: string[];
  score?: number;
};

export type EventWithPicks = Picks & {
  event: Event;
};
