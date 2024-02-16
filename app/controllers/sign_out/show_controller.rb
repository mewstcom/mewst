# typed: true
# frozen_string_literal: true

class SignOut::ShowController < ApplicationController
  include ControllerConcerns::Authenticatable
  include ControllerConcerns::Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    sign_out

    flash[:notice] = t("messages.authentication.signed_out_successfully")
    redirect_to root_path
  end
end
