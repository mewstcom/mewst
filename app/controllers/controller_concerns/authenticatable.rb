# typed: strict
# frozen_string_literal: true

module ControllerConcerns::Authenticatable
  extend T::Sig
  extend ActiveSupport::Concern

  included do
    helper_method :current_actor, :current_actor!, :signed_in?
  end

  sig(:final) { params(session: Session).returns(T::Boolean) }
  def sign_in(session)
    cookies.signed.permanent[Session::COOKIE_KEY] = {
      value: session.token,
      httponly: true,
      same_site: :lax
    }
    true
  end

  sig(:final) { returns(T::Boolean) }
  def sign_out
    cookies.delete(Session::COOKIE_KEY)
    true
  end

  sig(:final) { returns(T.nilable(Actor)) }
  def current_actor
    @current_actor ||= T.let(begin
      return unless cookies.signed[Session::COOKIE_KEY]
      Session.find_by(token: cookies.signed[Session::COOKIE_KEY])&.actor
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
      redirect_to home_path
    end
  end

  sig(:final) { returns(T.nilable(String)) }
  def actual_remote_ip
    request.env["HTTP_CF_CONNECTING_IP"] || request.remote_ip
  end
end
