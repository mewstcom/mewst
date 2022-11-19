# typed: strict
# frozen_string_literal: true

module Authenticatable
  extend T::Sig
  extend ActiveSupport::Concern

  included do
    helper_method :current_account, :current_profile, :signed_in?
  end

  sig { returns(T.nilable(Account)) }
  def current_account
    return unless session[:account_id]

    @current_account ||= T.let(Account.find_by(id: session[:account_id]), T.nilable(Account))
  end

  sig { returns(T.nilable(Profile)) }
  def current_profile
    unless session[:profile_id]
      return current_account.users.order(:id).first.profile
    end

    @current_profile ||= T.let(Profile.only_kept.find_by(id: session[:profile_id]), T.nilable(Profile))
  end

  sig { returns(T::Boolean) }
  def signed_in?
    current_account.present?
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
