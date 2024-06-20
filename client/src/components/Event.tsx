import { Event } from "../types";

interface EventDisplayProps {
  event: Event;
  picks: string[];
  onClickFighter: (fighter: string, opponent: string) => void;
}

function EventDisplay({ event, picks, onClickFighter }: EventDisplayProps) {
  const startTime = new Date(event.start_time);
  return (
    <div className="flex flex-col h-full items-stretch">
      <div>
        <h1 className="text-4xl text-center">Event {event.id}</h1>
        <h2 className="text-xl text-center">
          {startTime.toDateString()} at {startTime.toLocaleTimeString()}
        </h2>
      </div>
      <div className="flex-grow flex flex-col justify-center items-center">
        {event.fights.map((fight, index) => (
          <div
            key={index}
            className="flex items-center space-x-2 py-1 my-0.5 w-7/12 border-4 border-black bg-slate-400"
          >
            <div className="flex justify-evenly w-full gap-2 items-center px-4">
              <button
                onClick={() =>
                  onClickFighter(fight.fighters[0], fight.fighters[1])
                }
                className={`flex-1 ${
                  picks.includes(fight.fighters[0])
                    ? "bg-green-500 hover:bg-green-600"
                    : "bg-slate-500 hover:bg-slate-600"
                } p-2 rounded-lg font-bold`}
              >
                {fight.fighters[0]}
              </button>
              <p className="flex-1 text-center text-xl font-bold">vs</p>
              <button
                onClick={() =>
                  onClickFighter(fight.fighters[1], fight.fighters[0])
                }
                className={`flex-1 ${
                  picks.includes(fight.fighters[1])
                    ? "bg-green-500 hover:bg-green-600"
                    : "bg-slate-500 hover:bg-slate-600"
                } p-2 rounded-lg font-bold`}
              >
                {fight.fighters[1]}
              </button>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

export default EventDisplay;
