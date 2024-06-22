import { Event } from "../types";

interface EventDisplayProps {
  event: Event;
  picks: string[];
  onClickFighter: (fighter: string, opponent: string) => void;
  score?: number;
}

function EventDisplay({
  event,
  picks,
  onClickFighter,
  score,
}: EventDisplayProps) {
  const startTime = new Date(event.start_time);
  return (
    <div className="flex flex-col h-full">
      <div className="flex flex-col items-center">
        <h1 className="text-4xl font-bold">Event {event.id}</h1>
        <div className="flex flex-row w-7/12 items-center justify-between">
          {event.start_time === "LIVE" ? (
            <p className="text-2xl font-bold text-red-500">LIVE 🔴</p>
          ) : (
            <p className="text-xl">
              {startTime.toDateString() +
                (startTime > new Date()
                  ? ` at ${startTime.toLocaleTimeString()}`
                  : "")}
            </p>
          )}
          {score !== undefined && <p className="text-xl">Score: {score}</p>}
        </div>
      </div>
      <div className="flex-grow flex flex-col items-center">
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
                {fight.fighters[0] +
                  (fight.winner === fight.fighters[0] ? " 🏆" : "")}
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
                {fight.fighters[1] +
                  (fight.winner === fight.fighters[1] ? " 🏆" : "")}
              </button>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

export default EventDisplay;
