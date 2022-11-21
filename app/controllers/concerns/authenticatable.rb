# typed: strict
# frozen_string_literal: true

module Authenticatable
  extend T::Sig
  extend ActiveSupport::Concern

  included do
    helper_method :current_user, :signed_in?
  end

  sig { params(user: User).returns(Integer) }
  def sign_in(user)
    user.account.track_sign_in
    session[:user_id] = user.id
  end

  sig { returns(T.nilable(User)) }
  def current_user
    return unless session[:user_id]

    @current_user ||= T.let(User.find_by(id: session[:user_id]), T.nilable(User))
  end

  sig { returns(T::Boolean) }
  def signed_in?
    current_user.present?
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
