package outbox

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	"GolangTemplateProject/internal/adapters/primary/k8s"
	"GolangTemplateProject/internal/adapters/primary/prometheus"
	"GolangTemplateProject/internal/config"
	"GolangTemplateProject/internal/domain"
	"GolangTemplateProject/internal/ports"
	user_repo "GolangTemplateProject/internal/repository/user"
	"GolangTemplateProject/internal/usecase/outbox"
	"GolangTemplateProject/pkg/adapters/kafka"
	"GolangTemplateProject/pkg/adapters/kafka/consumer/dql"
	"GolangTemplateProject/pkg/adapters/postgres"
	"GolangTemplateProject/pkg/closer"
	"GolangTemplateProject/pkg/logger"
	"GolangTemplateProject/pkg/smart-span/tracing"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
)

type Application struct {
	postgres       postgres.IPostgres
	consumer       kafka.Consumer
	gracefulCloser *closer.GracefulCloser
	scheduler      jobs.Scheduler
	tracerProvider trace.TracerProvider
	logger         logger.Logger
}

func NewApplication(ctx context.Context) (*Application, error) {
	app := new(Application)
	err := app.SetupDependencies(ctx)
	if err != nil {
		return nil, err
	}
	return app, nil
}

func (a *Application) SetupDependencies(ctx context.Context) error {
	a.gracefulCloser = closer.NewGracefulCloser()
	err := config.LoadConfig()
	if err != nil {
		panic(err)
	}
	pool, err := postgres.New(ctx, &config.Get().Database.Postgres)
	a.postgres = pool
	if err != nil {
		panic(err)
	}
	a.gracefulCloser.AddCloser(pool.Close)

	if err != nil {
		panic(err)
	}
	a.gracefulCloser.AddCloser(a.scheduler.Stop)

	tracerProvider, f, err := tracing.InitTracing(ctx, &config.Get().Trace.Jaeger)
	if err != nil {
		panic(err)
	}
	a.gracefulCloser.AddCloser(func() error {
		return f(ctx)
	})
	a.logger, err = logger.NewLogger(config.Get().Env, tracerProvider.Tracer(config.Get().ServiceName))
	logger.SetDefaultLogger(a.logger)
	if err != nil {
		panic(err)
	}
	a.tracerProvider = tracerProvider
	return nil
}

func (a *Application) Start() {
	//ctx, cancel := context.WithCancel(context.Background())

	//producer, err := sync_producer.NewTopicProducer(config.Get().Kafka, nil)
	//fmt.Println(config.Get().Kafka)
	//if err != nil {
	//	panic(err)
	//}

	//trxManager := transaction_manager.New(a.postgres, a.logger)

	baseUserRepository := ports.NewBaseRepository[*domain.User](a.postgres, "user_registration")
	//baseUserSecretRepository := ports.NewBaseRepository[*domain.UserSecrets](a.postgres, "user_secret", func() *domain.UserSecrets {
	//	return &domain.UserSecrets{}
	//})
	dqlRepository := ports.NewBaseRepository[*domain.DLQMessage](a.postgres, domain.DlqMessageTable)
	userRepository := user_repo.NewUserRepository(baseUserRepository)
	//userSecretRepository := user_secret_repo.NewUserSecretRepository(baseUserSecretRepository)
	usecaseUser := outbox.NewUserUsecase(userRepository, nil)

	groupDlq, err := dql.NewTopicConsumerGroupDlq[*domain.User](
		&config.Get().KafkaConsumer,
		logger.DefaultLogger(),
		dqlRepository.Create,
		usecaseUser.SaveUser,
		dql.WithDLQBatchSaver(dqlRepository.CreateBatch),
	)
	if err != nil {
		return
	}

	engine := gin.Default()

	k8s.NewModulePrometheus().Register(engine)
	prometheus.NewModulePromehteus().Register(engine.Group("/metrics"))
	//userHandlers := auth_handlers.NewUserHandlers(usecaseUser)
	//
	//grpcServer := grpc.NewServer()
	//user_v1.RegisterAuthServiceServer(grpcServer, userHandlers)
	//listener, err := net.Listen("tcp", config.Get().ServerInfo.HttpConnection.String())
	//if err != nil {
	//	panic(err)
	//}
	//go func() {
	//	if err := grpcServer.Serve(listener); err != nil {
	//		panic(err)
	//	}
	//}()

	//// Todo : дополнить конфиг
	//proxy := server_grpc.NewProxy(server_grpc.ServerConfig{})
	//proxy.AddService(user_v1.RegisterAuthServiceHandlerFromEndpoint)
	//
	//srv := &http.Server{
	//	Addr:    config.Get().ServerInfo.HttpConnection.String(),
	//	Handler: engine,
	//}

	//go func() {
	//	fmt.Printf("Service started on %s", srv.Addr)
	//	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
	//		log.Fatalf("listen: %s\n", err)
	//	}
	//}()
	//
	//a.gracefulCloser.AddCloser(func() error {
	//	ctxWithTimeout, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	//	defer cancelFunc()
	//	return srv.Shutdown(ctxWithTimeout)
	//})

	a.scheduler.Start()
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	<-interrupt
	log.Println("shutting down...")
	if err := a.gracefulCloser.Close(); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Println("forced shutdown: some components didn't stop in time")
		} else {
			log.Println("shutdown error:", err)
		}
	}
	//cancel()
}
