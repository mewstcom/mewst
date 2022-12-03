# typed: true
# frozen_string_literal: true

class Users::CreateController < ApplicationController
  include Authenticatable

  before_action :require_phone_number_confirmation_id
  before_action :require_no_authentication

  sig { returns(T.untyped) }
  def call
    @phone_number_confirmation = PhoneNumberConfirmation.find(session[:phone_number_confirmation_id])
    @form = NewUserForm.new(form_params.merge(phone_number_confirmation: @phone_number_confirmation))

    if @form.invalid?
      return render("users/new/call")
    end

    result = CreateUserService.new(form: @form).call

    reset_session
    sign_in(T.must(result.user))

    redirect_to home_path
  end

  sig { returns(ActionController::Parameters) }
  private def form_params
    T.cast(params.require(:new_user_form), ActionController::Parameters).permit(:idname)
  end

  sig { returns(T.untyped) }
  private def require_phone_number_confirmation_id
    unless session[:phone_number_confirmation_id]
      redirect_to root_path
    end
  end
end
