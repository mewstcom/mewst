# typed: true
# frozen_string_literal: true

class Accounts::CreateController < ApplicationController
  include Authenticatable

  before_action :require_no_authentication

  sig { returns(T.untyped) }
  def call
    @form = SignUpForm.new(form_params)

    if @form.invalid?
      return render("accounts/new/call")
    end

    result = ActiveRecord::Base.transaction do
      SignUpService.new(form: @form).call
    end

    reset_session
    sign_in(result.account)

    redirect_to home_path
  end

  sig { returns(ActionController::Parameters) }
  private def form_params
    T.cast(params.require(:sign_up_form), ActionController::Parameters).permit(:email, :idname)
  end
end
