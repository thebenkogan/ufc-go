import { useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { get } from "./api";

function AuthCallback() {
    const navigate = useNavigate();
    const path = "auth/google/callback" + window.location.search;
    
    useEffect(() => {
      get(path).then(() => {
          navigate("/");
      });
    }, [path, navigate]);
  
    return <div className="bg-blue-300">auth callback</div>;
  }
  
  export default AuthCallback;