APP_NAME = Orden_Ejecucion_Prog         
SRC = PruebaC2.go                       
GO = go                                
BUILD_FLAGS = -o $(APP_NAME)            
ARGS = 1 2 50 orden_creacion.txt salida.txt

all: build run

build:
	$(GO) build $(BUILD_FLAGS) $(SRC)

run: build
	./$(APP_NAME) $(ARGS)

clean:
	rm -f $(APP_NAME)

clean_all: clean
	$(GO) clean -cache -modcache -testcache