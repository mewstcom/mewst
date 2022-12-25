# typed: strict
# frozen_string_literal: true

module Authenticatable
  extend T::Sig
  extend ActiveSupport::Concern

  included do
    helper_method :current_profile, :current_profile!, :current_account, :current_account!, :signed_in?
  end

  sig { params(account: Account).returns(T::Boolean) }
  def sign_in(account)
    account.track_sign_in
    session[:account_id] = account.id
    true
  end

  sig { returns(T::Boolean) }
  def sign_out
    reset_session
    true
  end

  sig { returns(T.nilable(Account)) }
  def current_account
    return unless session[:account_id]

    @current_account ||= T.let(Account.find_by(id: session[:account_id]), T.nilable(Account))
  end

  sig { returns(Account) }
  def current_account!
    T.cast(current_account, Account)
  end

  sig { returns(T.nilable(Profile)) }
  def current_profile
    return unless signed_in?

    @current_profile ||= T.let(current_account!.profile, T.nilable(Profile))
  end

  sig { returns(Profile) }
  def current_profile!
    T.cast(current_profile, Profile)
  end

  sig { returns(T::Boolean) }
  def signed_in?
    !current_account.nil?
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
      flash[:success] = t("messages.authentication.already_signed_in")
      redirect_to root_path
    end
  end
end
