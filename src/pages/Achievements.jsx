import React, { useEffect, useState } from "react";
import Header from "../components/Header";
const achievements = [
  //* Completed from backend this is done when updating trophies and then incrementing the winning match count
  {
    id: 1,
    title: "First Victory",
    description: "Win your first 1v1 quiz match",
  },
  //* Completed in frontend
  {
    id: 2,
    title: "Perfect Round",
    description: "Answer all questions in a game correctly",
  },
  //* Completed in frontend
  {
    id: 3,
    title: "Lightning Reflexes",
    description: "Answer a question within 3 seconds",
  },
  //* Completed updating history in the backend -1,0,1 as lost,drawn and won respectively
  { id: 4, title: "Quiz Champion", description: "Win 10 consecutive matches" },
  // * Completed in backend
  {
    id: 5,
    title: "Clutch Performer",
    description: "Win a game after being 40 points behind",
  },
];

export default function Achievements() {
  // ! LOCAL STORAGE AS OF NOW
  const [profileName] = useState(localStorage.getItem("profileName"));
  const [achievementsData, setAchievementsData] = useState([]);
  useEffect(() => {
    const getAchievementData = async () => {
      const response = await fetch(
        `http://localhost:5000/get-achievement-data?profileName=${profileName}`
      );
      const data = await response.json();
      console.log(data);
      setAchievementsData(data.achievements);
    };
    getAchievementData();
  }, []);

  return (
    <div className="flex flex-col h-screen w-screen font-roboto overflow-hidden">
      <Header />
      {/* Correct y-axis space between topic and the next section */}
      <div className="bg-[#C5E6DF] text-black flex flex-col space-y-20 h-full w-full items-center bg-decoration">
        {/* Correct space of header from header section */}
        <h1 className="text-2xl tracking-widest mt-20 font-bebas-neue border-2 p-2 border-black pl-8 pr-8">
          Acheivements
        </h1>
        <ul className="space-y-4">
          {achievements &&
            achievements.map((achievement, index) => (
              <li
                key={achievement.id}
                className="flex items-center justify-between space-x-2"
              >
                <div className="flex flex-col">
                  <h1 className="text-lg text-teal-800 font-semibold">
                    {achievement.title}
                  </h1>
                  <p className="text-sm">{achievement.description}</p>
                </div>
                <div
                  className={`p-2 ${
                    achievementsData && achievementsData[index] === true
                      ? "bg-green-700"
                      : "bg-[#474747]"
                  }  rounded-2xl`}
                >
                  <img
                    src={`${achievement.title}.svg`}
                    alt={achievement.title}
                    height="16"
                    width="16"
                  />
                </div>
              </li>
            ))}
        </ul>
      </div>
    </div>
  );
}
