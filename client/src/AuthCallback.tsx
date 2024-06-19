import { useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { authCallback } from "./api";

function AuthCallback() {
  const navigate = useNavigate();

  useEffect(() => {
    authCallback().then(() => {
      navigate("/");
    });
  });

  return <div className="bg-blue-300">auth callback</div>;
}

export default AuthCallback;
