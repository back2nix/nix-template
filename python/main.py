import os
import sys
import numpy as np

def main():
    print(f"Hello from Python! version: {sys.version}")
    print(f"NumPy version: {np.__version__}")
    print(f"Current Directory: {os.getcwd()}")

    # Проверка переменных из .env (если python-dotenv установлен)
    try:
        from dotenv import load_dotenv
        load_dotenv()
        print("Dotenv loaded.")
    except ImportError:
        print("Dotenv not installed.")

if __name__ == "__main__":
    main()
