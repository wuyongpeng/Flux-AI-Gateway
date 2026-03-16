import requests
import time
import json
import threading

URL = "http://localhost:8080/v1/chat/completions"
PAYLOAD = {
    "contents": [{"parts": [{"text": "Explain quantum physics in 20 words."}]}]
}

def get_ttft(scenario_name):
    print(f"Testing {scenario_name}...")
    headers = {
        "Content-Type": "application/json",
        "X-Flux-Scenario": scenario_name
    }
    
    start_time = time.time()
    try:
        response = requests.post(URL, json=PAYLOAD, headers=headers, stream=True, timeout=30)
        ttft = None
        
        for line in response.iter_lines():
            if line:
                decoded_line = line.decode('utf-8')
                if "[Metrics] TTFT:" in decoded_line:
                    # Extract TTFT from SSE data line
                    ttft = decoded_line.split("TTFT:")[1].strip()
                    break
        
        print(f"[{scenario_name}] Winner Metric: {ttft}")
        return ttft
    except Exception as e:
        print(f"[{scenario_name}] Error: {e}")
        return None

if __name__ == "__main__":
    results = {}
    
    # Test GLM
    results['GLM'] = get_ttft("only_glm")
    
    # Test Gemini
    results['Gemini'] = get_ttft("only_gemini")
    
    print("\n" + "="*30)
    print("Performance Comparison")
    print("="*30)
    for provider, latency in results.items():
        print(f"{provider:10}: {latency}")
    print("="*30)
