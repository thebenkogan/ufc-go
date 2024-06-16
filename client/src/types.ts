export type Event = {
  id: string;
  start_time: string;
  fights: Fight[];
};

export type Fight = {
  fighters: string[];
  winner?: string;
};
