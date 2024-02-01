# typed: strict
# frozen_string_literal: true

module ControllerConcerns::Authenticatable
  extend T::Sig
  extend ActiveSupport::Concern

  included do
    helper_method :current_actor, :signed_in?
  end

  sig { params(actor: Actor).returns(T::Boolean) }
  def sign_in(actor)
    session[:current_actor_id] = actor.id
    true
  end

  sig { returns(T::Boolean) }
  def sign_out
    reset_session
    true
  end

  sig { returns(T.nilable(Actor)) }
  def current_actor
    return unless session[:current_actor_id]

    @current_actor ||= T.let(begin
      actor = Actor.new(id: session[:current_actor_id])
      actor.persisted? ? actor : nil
    end, T.nilable(Actor))
  end

  sig { returns(T::Boolean) }
  def signed_in?
    !current_actor.nil?
  end

  sig { returns(T.untyped) }
  def require_authentication
    unless signed_in?
      redirect_to root_path
    end
  end

  sig { returns(T.untyped) }
  def require_no_authentication
    if signed_in?
      flash[:notice] = t("messages.authentication.already_signed_in")
      redirect_to root_path
    end
  end
end
