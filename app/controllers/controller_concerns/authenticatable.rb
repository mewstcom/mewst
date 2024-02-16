# typed: strict
# frozen_string_literal: true

module ControllerConcerns::Authenticatable
  extend T::Sig
  extend ActiveSupport::Concern

  included do
    helper_method :current_actor, :current_actor!, :signed_in?
  end

  sig(:final) { params(actor: Actor).returns(T::Boolean) }
  def sign_in(actor)
    session[:current_actor_id] = actor.id
    true
  end

  sig(:final) { returns(T::Boolean) }
  def sign_out
    reset_session
    true
  end

  sig(:final) { returns(T.nilable(Actor)) }
  def current_actor
    @current_actor ||= T.let(begin
      return unless session[:current_actor_id]
      Actor.find_by(id: session[:current_actor_id])
    end, T.nilable(Actor))
  end

  sig(:final) { returns(Actor) }
  def current_actor!
    current_actor.not_nil!
  end

  sig(:final) { returns(T::Boolean) }
  def signed_in?
    !current_actor.nil?
  end

  sig(:final) { returns(T.untyped) }
  def require_authentication
    unless signed_in?
      redirect_to root_path
    end
  end

  sig(:final) { returns(T.untyped) }
  def require_no_authentication
    if signed_in?
      flash[:notice] = t("messages.authentication.already_signed_in")
      redirect_to root_path
    end
  end
end
