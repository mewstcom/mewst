# typed: strict
# frozen_string_literal: true

module ControllerConcerns::Authenticatable
  extend T::Sig
  extend ActiveSupport::Concern

  included do
    helper_method :signed_in?, :viewer, :viewer!
  end

  sig(:final) { params(session: SessionRecord).returns(T::Boolean) }
  def sign_in(session)
    cookies.signed.permanent[SessionRecord::COOKIE_KEY] = {
      value: session.token,
      httponly: true,
      same_site: :lax
    }
    true
  end

  sig(:final) { returns(T::Boolean) }
  def sign_out
    cookies.delete(SessionRecord::COOKIE_KEY)
    true
  end

  sig(:final) { returns(T.nilable(ActorRecord)) }
  def viewer
    @viewer ||= T.let(begin
      return unless cookies.signed[SessionRecord::COOKIE_KEY]
      SessionRecord.find_by(token: cookies.signed[SessionRecord::COOKIE_KEY])&.actor
    end, T.nilable(ActorRecord))
  end

  sig(:final) { returns(ActorRecord) }
  def viewer!
    viewer.not_nil!
  end

  sig(:final) { returns(T::Boolean) }
  def signed_in?
    !viewer.nil?
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
  def original_remote_ip
    request.env["HTTP_CF_CONNECTING_IP"] || request.remote_ip
  end
end
