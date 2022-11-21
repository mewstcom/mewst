# typed: true
# frozen_string_literal: true

class SignOut::ShowController < ApplicationController
  include Authenticatable

  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    sign_out

    flash[:success] = t("messages.authentication.signed_out_successfully")
    redirect_to root_path
  end
end
