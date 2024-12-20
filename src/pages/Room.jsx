import { useEffect, useState } from "react";
import { useWebSocket } from "../contexts/WebSocketContext"; // Adjust the path as necessary
import Header from "../components/Header";

const Room = () => {
  const url = new URLSearchParams(window.location.search);
  const { ws } = useWebSocket();
  const [questions, setQuestions] = useState([]);
  const [currentQuestionIndex, setCurrentQuestionIndex] = useState(0);
  const [totalPoints, setTotalPoints] = useState(0);
  const [questionTimer, setQuestionTimer] = useState(5);
  const [isMatchCompleted, setIsMatchCompleted] = useState(false);

  const [roomId] = useState(url.get("id"));
  const [profileName] = useState(url.get("profileName"));
  const [opponentTotalPoints, setOpponentTotalPoints] = useState(null);
  const [matchResult, setMatchResult] = useState(null);
  useEffect(() => {
    if (ws) {
      // Handle incoming messages or perform actions
      ws.onmessage = (e) => {
        const data = JSON.parse(e.data);
        console.log("Message received in Room:", data);
        // We received opponent total points
        setOpponentTotalPoints(data.opponent_total_points);
      };
    }
    return () => {
      // Optional cleanup if needed
    };
  }, [ws]);

  useEffect(() => {
    // const fetchQuestions = async () => {
    //   const response = await fetch(
    //     "https://opentdb.com/api.php?amount=10&category=21&type=multiple"
    //   );
    //   const data = await response.json();
    //   setQuestions
    // };
    // fetchQuestions();
    setQuestions([
      {
        type: "multiple",
        difficulty: "easy",
        category: "Sports",
        question: "In baseball, how many fouls are an out?",
        correct_answer: "0",
        incorrect_answers: ["5", "3", "2"],
      },
      {
        type: "multiple",
        difficulty: "medium",
        category: "Sports",
        question:
          "Which NBA player won Most Valuable Player for the 1999-2000 season?",
        correct_answer: "Shaquille O&#039;Neal",
        incorrect_answers: ["Allen Iverson", "Kobe Bryant", "Paul Pierce"],
      },
      {
        type: "multiple",
        difficulty: "easy",
        category: "Sports",
        question: "What team won the 2016 MLS Cup?",
        correct_answer: "Seattle Sounders",
        incorrect_answers: ["Colorado Rapids", "Toronto FC", "Montreal Impact"],
      },
      {
        type: "multiple",
        difficulty: "medium",
        category: "Sports",
        question:
          "What is the exact length of one non-curved part in Lane 1 of an Olympic Track?",
        correct_answer: "84.39m",
        incorrect_answers: ["100m", "100yd", "109.36yd"],
      },
      {
        type: "multiple",
        difficulty: "medium",
        category: "Sports",
        question:
          "Which of the following player scored a hat-trick during their Manchester United debut?",
        correct_answer: "Wayne Rooney",
        incorrect_answers: [
          "Cristiano Ronaldo",
          "Robin Van Persie",
          "David Beckham",
        ],
      },
    ]);
  }, []);

  // Run side-effects on question timer on first render and when the question changes
  useEffect(() => {
    const intervalId = setInterval(() => {
      // Question timer starts from 5 if the timer count is not 0 then decrement timer by 1 else if the timer is 0 then reset question timer to 5
      setQuestionTimer((prevTimer) => {
        if (prevTimer === 0) {
          setQuestionTimer(5);
          // Increment the question index
          setCurrentQuestionIndex((prevIndex) => {
            const nextIndex = prevIndex + 1;
            if (nextIndex < questions.length) {
              return nextIndex;
            }
            // Clear the previously created interval
            clearInterval(intervalId);
            return prevIndex;
          });
        }
        return prevTimer - 1;
      });
    }, 1000);

    return () => clearInterval(intervalId);
  }, [questions.length]);

  const handleOptionClick = (selectedOption) => {
    const currentQuestion = questions[currentQuestionIndex];
    // Tracking points locally
    const newTotalPoints =
      selectedOption === currentQuestion.correct_answer
        ? totalPoints + 20
        : totalPoints;
    setTotalPoints(newTotalPoints);

    // Move to the next question
    if (currentQuestionIndex < questions.length - 1) {
      setCurrentQuestionIndex(currentQuestionIndex + 1);
    } else {
      ws.send(
        JSON.stringify({
          action: "player_completed",
          roomId: roomId,
          profileName: profileName,
          playerPoints: newTotalPoints,
        })
      );
      // Marking match as completed
      setIsMatchCompleted(true);
    }
  };
  // Run sideeffect whenever the match is completed and when we get the opponent's points
  useEffect(() => {
    // You have to complete the match and also need the opponent's total points
    if (isMatchCompleted && opponentTotalPoints) {
      if (totalPoints > opponentTotalPoints) {
        setMatchResult("won");
      } else if (totalPoints < opponentTotalPoints) {
        setMatchResult("lost");
      } else {
        setMatchResult("tie");
      }
      // Store the points scored in the quiz
      // Ending the match removing the players from server
      ws.send(JSON.stringify({ action: "match_completed", roomId: roomId }));
      // Close the websocket server
      ws.close();
    }
  }, [isMatchCompleted, opponentTotalPoints]);
  return (
    <div className="flex flex-col h-screen w-screen text-white font-roboto">
      <Header />
      <div className="flex flex-col h-full w-full bg-[#C5E6DF] text-black items-center justify-center">
        {/* Render the current question */}
        {!isMatchCompleted ? (
          questions.length > 0 && (
            <div className="flex flex-col items-start text-sm space-y-4">
              {/* Clock timer for question */}
              {questionTimer}
              {/* Question */}
              {/* Highest Width so this is the decider to items center / justify center */}
              <p className="w-60">
                {currentQuestionIndex + 1}.{" "}
                {questions[currentQuestionIndex].question}
              </p>
              {/* Options */}
              <div className="flex flex-col space-y-4">
                {/* Include both correct and incorrect answers and shuffle them */}
                {[
                  ...questions[currentQuestionIndex].incorrect_answers,
                  questions[currentQuestionIndex].correct_answer,
                ].map((option, index) => (
                  <button
                    key={index}
                    onClick={() => handleOptionClick(option)}
                    className="p-4 text-left bg-gray-800 text-white rounded-3xl hover:bg-green-400 duration-300"
                  >
                    <span>{index + 1}.</span> {option}
                  </button>
                ))}
              </div>
            </div>
          )
        ) : (
          <h1 className="text-2xl">
            <p>Your Points: {totalPoints}</p>
            <p>
              Opponent Points:{" "}
              {opponentTotalPoints !== null
                ? opponentTotalPoints
                : "Loading..."}
            </p>
          </h1>
        )}
        {matchResult === "won" ? (
          <div className="absolute h-fit w-screen flex items-center justify-center bg-green-600">
            <h1 className="text-6xl text-white">You Won!</h1>
          </div>
        ) : matchResult === "lost" ? (
          <div className="absolute h-fit w-screen flex items-center justify-center bg-red-600">
            <h1 className="text-6xl text-white">You Lost!</h1>
          </div>
        ) : matchResult === "tie" ? (
          <div className="absolute h-fit w-screen flex items-center justify-center bg-yellow-600">
            <h1 className="text-6xl text-white">It's a Tie!</h1>
          </div>
        ) : null}
      </div>
    </div>
  );
};

export default Room;
