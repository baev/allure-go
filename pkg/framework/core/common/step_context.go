package common

import (
	"fmt"
	"runtime/debug"
	"sync"
	"testing"

	"github.com/ozontech/allure-go/pkg/allure"
	"github.com/ozontech/allure-go/pkg/framework/asserts_wrapper/helper"
	"github.com/ozontech/allure-go/pkg/framework/provider"
)

type internalStepCtx interface {
	provider.StepCtx

	ExecutionContextName() string
	WG() *sync.WaitGroup
}

type stepCtx struct {
	t testing.TB
	p provider.Provider

	currentStep *allure.Step
	parentStep  internalStepCtx

	asserts provider.Asserts
	require provider.Asserts

	wg sync.WaitGroup
}

func newStepCtx(t testing.TB, p provider.Provider, stepName string, params ...allure.Parameter) internalStepCtx {
	currentStep := allure.NewSimpleStep(stepName, params...)
	newCtx := &stepCtx{t: t, p: p, currentStep: currentStep, wg: sync.WaitGroup{}}
	newCtx.asserts = helper.NewAssertsHelper(newCtx)
	newCtx.require = helper.NewRequireHelper(newCtx)
	return newCtx
}

func (ctx *stepCtx) newChildCtx(stepName string, params ...allure.Parameter) internalStepCtx {
	currentStep := allure.NewSimpleStep(stepName, params...)
	newCtx := &stepCtx{t: ctx.t, p: ctx.p, currentStep: currentStep, parentStep: ctx, wg: sync.WaitGroup{}}
	newCtx.asserts = helper.NewAssertsHelper(newCtx)
	newCtx.require = helper.NewRequireHelper(newCtx)
	return newCtx
}

func (ctx *stepCtx) Assert() provider.Asserts {
	return ctx.asserts
}

func (ctx *stepCtx) Require() provider.Asserts {
	return ctx.require
}

func (ctx *stepCtx) WG() *sync.WaitGroup {
	return &ctx.wg
}

func (ctx *stepCtx) ExecutionContextName() string {
	return ctx.p.ExecutionContext().GetName()
}

func (ctx *stepCtx) FailNow() {
	ctx.t.FailNow()
}

func (ctx *stepCtx) Error(args ...interface{}) {
	ctx.Fail()
	ctx.t.Error(args...)
}

func (ctx *stepCtx) Errorf(format string, args ...interface{}) {
	ctx.Fail()
	ctx.t.Errorf(format, args...)
}

func (ctx *stepCtx) Log(args ...interface{}) {
	ctx.t.Log(args...)
}

func (ctx *stepCtx) Logf(format string, args ...interface{}) {
	ctx.t.Logf(format, args...)
}

func (ctx *stepCtx) CurrentStep() *allure.Step {
	return ctx.currentStep
}

func (ctx *stepCtx) WithParameters(parameters ...allure.Parameter) {
	ctx.currentStep.WithParameters(parameters...)
}

func (ctx *stepCtx) WithNewParameters(kv ...string) {
	ctx.currentStep.WithNewParameters(kv...)
}

func (ctx *stepCtx) WithAttachments(attachments ...*allure.Attachment) {
	ctx.currentStep.WithAttachments(attachments...)
}

func (ctx *stepCtx) WithNewAttachment(name string, mimeType allure.MimeType, content []byte) {
	ctx.currentStep.WithAttachments(allure.NewAttachment(name, mimeType, content))
}

func (ctx *stepCtx) Step(step *allure.Step) {
	ctx.currentStep.WithChild(step)
}

func (ctx *stepCtx) NewStep(stepName string, parameters ...allure.Parameter) {
	newStep := allure.NewSimpleStep(stepName, parameters...)
	ctx.currentStep.WithChild(newStep)
}

func (ctx *stepCtx) WithNewStep(stepName string, step func(ctx provider.StepCtx), params ...allure.Parameter) {
	newCtx := ctx.newChildCtx(stepName, params...)
	defer ctx.currentStep.WithChild(newCtx.CurrentStep())
	defer func() {
		r := recover()
		if r != nil {
			ctxName := newCtx.ExecutionContextName()
			errMsg := fmt.Sprintf("%s panicked: %v\n%s", ctxName, r, debug.Stack())
			newCtx.Broken()
			TestError(ctx.t, ctx.p, ctx.p.ExecutionContext().GetName(), errMsg)
		}
	}()
	step(newCtx)
}

func (ctx *stepCtx) WithNewAsyncStep(stepName string, step func(ctx provider.StepCtx), params ...allure.Parameter) {
	var wg *sync.WaitGroup
	wg = &ctx.wg
	if ctx.parentStep != nil {
		wg = ctx.parentStep.WG()
		defer wg.Wait()
	}
	wg.Add(1)

	go func() {
		defer wg.Done()
		ctx.WithNewStep(stepName, step, params...)
	}()
}

func (ctx *stepCtx) Fail() {
	ctx.currentStep.Failed()
	if ctx.parentStep != nil {
		ctx.parentStep.Fail()
	}
}

func (ctx *stepCtx) Broken() {
	ctx.currentStep.Broken()
	if ctx.parentStep != nil {
		ctx.parentStep.Broken()
	}
}
