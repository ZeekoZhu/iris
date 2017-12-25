package mvc

import (
	"github.com/kataras/iris/core/router"
	"github.com/kataras/iris/hero/di"
)

// Application is the high-level compoment of the "mvc" package.
// It's the API that you will be using to register controllers among with their
// dependencies that your controllers may expecting.
// It contains the Router(iris.Party) in order to be able to register
// template layout, middleware, done handlers as you used with the
// standard Iris APIBuilder.
//
// The Engine is created by the `New` method and it's the dependencies holder
// and controllers factory.
//
// See `mvc#New` for more.
type Application struct {
	Dependencies di.Values
	Router       router.Party
}

func newApp(subRouter router.Party, values di.Values) *Application {
	return &Application{
		Router:       subRouter,
		Dependencies: values,
	}
}

// New returns a new mvc Application based on a "party".
// Application creates a new engine which is responsible for binding the dependencies
// and creating and activating the app's controller(s).
//
// Example: `New(app.Party("/todo"))` or `New(app)` as it's the same as `New(app.Party("/"))`.
func New(party router.Party) *Application {
	return newApp(party, di.NewValues())
}

// Configure creates a new controller and configures it,
// this function simply calls the `New(party)` and its `.Configure(configurators...)`.
//
// A call of `mvc.New(app.Party("/path").Configure(buildMyMVC)` is equal to
//           	 `mvc.Configure(app.Party("/path"), buildMyMVC)`.
//
// Read more at `New() Application` and `Application#Configure` methods.
func Configure(party router.Party, configurators ...func(*Application)) *Application {
	// Author's Notes->
	// About the Configure's comment: +5 space to be shown in equal width to the previous or after line.
	//
	// About the Configure's design chosen:
	// Yes, we could just have a `New(party, configurators...)`
	// but I think the `New()` and `Configure(configurators...)` API seems more native to programmers,
	// at least to me and the people I ask for their opinion between them.
	// Because the `New()` can actually return something that can be fully configured without its `Configure`,
	// its `Configure` is there just to design the apps better and help end-devs to split their code wisely.
	return New(party).Configure(configurators...)
}

// Configure can be used to pass one or more functions that accept this
// Application, use this to add dependencies and controller(s).
//
// Example: `New(app.Party("/todo")).Configure(func(mvcApp *mvc.Application){...})`.
func (app *Application) Configure(configurators ...func(*Application)) *Application {
	for _, c := range configurators {
		c(app)
	}
	return app
}

// AddDependencies adds one or more values as dependencies.
// The value can be a single struct value-instance or a function
// which has one input and one output, the input should be
// an `iris.Context` and the output can be any type, that output type
// will be binded to the controller's field, if matching or to the
// controller's methods, if matching.
//
// These dependencies "values" can be changed per-controller as well,
// via controller's `BeforeActivation` and `AfterActivation` methods,
// look the `Register` method for more.
//
// It returns this Application.
//
// Example: `.AddDependencies(loggerService{prefix: "dev"}, func(ctx iris.Context) User {...})`.
func (app *Application) AddDependencies(values ...interface{}) *Application {
	app.Dependencies.Add(values...)
	return app
}

// Register adds a controller for the current Router.
// It accept any custom struct which its functions will be transformed
// to routes.
//
// If "controller" has `BeforeActivation(b mvc.BeforeActivation)`
// or/and `AfterActivation(a mvc.AfterActivation)` then these will be called between the controller's `.activate`,
// use those when you want to modify the controller before or/and after
// the controller will be registered to the main Iris Application.
//
// It returns this mvc Application.
//
// Usage: `.Register(new(TodoController))`.
//
// Controller accepts a sub router and registers any custom struct
// as controller, if struct doesn't have any compatible methods
// neither are registered via `ControllerActivator`'s `Handle` method
// then the controller is not registered at all.
//
// A Controller may have one or more methods
// that are wrapped to a handler and registered as routes before the server ran.
// The controller's method can accept any input argument that are previously binded
// via the dependencies or route's path accepts dynamic path parameters.
// The controller's fields are also bindable via the dependencies, either a
// static value (service) or a function (dynamically) which accepts a context
// and returns a single value (this type is being used to find the relative field or method's input argument).
//
// func(c *ExampleController) Get() string |
// (string, string) |
// (string, int) |
// int |
// (int, string |
// (string, error) |
// bool |
// (any, bool) |
// error |
// (int, error) |
// (customStruct, error) |
// customStruct |
// (customStruct, int) |
// (customStruct, string) |
// Result or (Result, error)
// where Get is an HTTP Method func.
//
// Examples at: https://github.com/kataras/iris/tree/master/_examples/mvc
func (app *Application) Register(controller interface{}) *Application {
	// initialize the controller's activator, nothing too magical so far.
	c := newControllerActivator(app.Router, controller, app.Dependencies)

	// check the controller's "BeforeActivation" or/and "AfterActivation" method(s) between the `activate`
	// call, which is simply parses the controller's methods, end-dev can register custom controller's methods
	// by using the BeforeActivation's (a ControllerActivation) `.Handle` method.
	if before, ok := controller.(interface {
		BeforeActivation(BeforeActivation)
	}); ok {
		before.BeforeActivation(c)
	}

	c.activate()

	if after, okAfter := controller.(interface {
		AfterActivation(AfterActivation)
	}); okAfter {
		after.AfterActivation(c)
	}
	return app
}

// NewChild creates and returns a new MVC Application which will be adapted
// to the "party", it adopts
// the parent's (current) dependencies, the "party" may be
// a totally new router or a child path one via the parent's `.Router.Party`.
//
// Example: `.NewChild(irisApp.Party("/path")).Register(new(TodoSubController))`.
func (app *Application) NewChild(party router.Party) *Application {
	return newApp(party, app.Dependencies.Clone())
}
