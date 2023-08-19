package game

import (
	"github.com/memmaker/terminal-assassin/console"
	"github.com/memmaker/terminal-assassin/game/services"
)

type Animator struct {
	engine             services.Engine
	currentAnimationID uint64
	animationsRunning  map[uint64]*ActiveAnimation
	particles          []services.Particle
}

func (a *Animator) AddParticle(particle services.Particle) {
	a.particles = append(a.particles, particle)
}
func (a *Animator) ClearParticles() {
	a.particles = []services.Particle{}
}
func NewAnimator(engine services.Engine) *Animator {
	return &Animator{
		engine:             engine,
		currentAnimationID: 0,
		animationsRunning:  make(map[uint64]*ActiveAnimation),
	}
}
func (a *Animator) advanceFrame(animationId uint64) {
	var anim *ActiveAnimation
	if _, ok := a.animationsRunning[animationId]; !ok {
		return
	}

	anim = a.animationsRunning[animationId]

	anim.CurrentFrame = anim.CurrentFrame + 1
	anim.TicksAliveForCurrentFrame = 0
	if (anim.FinishCondition != nil && anim.FinishCondition()) || (anim.frameCount > 0 && anim.CurrentFrame >= anim.frameCount) {
		delete(a.animationsRunning, animationId)
		if anim.FinishedCallback != nil {
			anim.FinishedCallback()
		}
		return
	}

	if anim.Update != nil {
		anim.Update(anim.CurrentFrame)
		return
	}
}

func (a *Animator) addAnimation(anim *ActiveAnimation) uint64 {
	a.currentAnimationID++
	animID := a.currentAnimationID
	anim.ID = animID
	anim.CurrentFrame = 0
	a.animationsRunning[animID] = anim

	if anim.Update != nil { // update the first frame
		anim.Update(anim.CurrentFrame)
	}
	return animID
}

func (a *Animator) drawAnimations(con console.CellInterface) {
	for _, anim := range a.animationsRunning {
		if anim.DrawFrame != nil { //&& anim.isDirty {
			anim.DrawFrame(con, anim.CurrentFrame)
			//anim.isDirty = false
		}
	}
}

func (a *Animator) drawParticles(con console.CellInterface) {
	for _, particle := range a.particles {
		particle.Draw(con)
	}
}

func (a *Animator) updateAnimations() {
	for animID, anim := range a.animationsRunning {
		anim.TicksAliveForCurrentFrame++
		if anim.CancelCondition != nil && anim.CancelCondition() {
			delete(a.animationsRunning, animID)
			if anim.CancelCallback != nil {
				anim.CancelCallback() // should we call finish on a cancel event?
			}
			continue
		}
		if anim.NextFrame() {
			a.advanceFrame(animID)
		}
	}
}

func (a *Animator) updateParticles() {
	if len(a.particles) == 0 {
		return
	}
	for i := len(a.particles) - 1; i >= 0 && len(a.particles) > 0; i-- {
		if a.particles[i].IsDead() {
			a.particles = append(a.particles[:i], a.particles[i+1:]...)
		} else {
			a.particles[i].Update(a.engine)
		}
	}
}

func (a *Animator) Update() {
	a.updateAnimations()
	a.updateParticles()
	/*
		if len(a.animationsRunning) > 0 {
			println("Animations running: ", len(a.animationsRunning))
		}
		if len(a.particles) > 0 {
			println("Particles: ", len(a.particles))
		}
	*/
}

func (a *Animator) Draw(con console.CellInterface) {
	a.drawAnimations(con)
	a.drawParticles(con)
}

func (a *Animator) ConfirmOrCancel() bool {
	return a.engine.GetInput().ConfirmOrCancel()
}

func (a *Animator) Reset() {
	a.animationsRunning = make(map[uint64]*ActiveAnimation)
	a.particles = []services.Particle{}
	a.currentAnimationID = 0
}
