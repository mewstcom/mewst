# typed: true
# frozen_string_literal: true

class Users::CreateController < ApplicationController
  include Authenticatable

  before_action :require_confirmed_phone_number
  before_action :require_no_authentication

  sig { returns(T.untyped) }
  def call
    @phone_number = PhoneNumber.find(session[:phone_number_id])
    @form = NewUserForm.new(form_params.merge(phone_number: @phone_number))

    if @form.invalid?
      return render("users/new/call")
    end

    result = CreateUserService.new(form: @form).call

    reset_session
    sign_in(result.user)

    redirect_to home_path
  end

  sig { returns(ActionController::Parameters) }
  private def form_params
    T.cast(params.require(:new_user_form), ActionController::Parameters).permit(:idname)
  end

  sig { returns(T.untyped) }
  private def require_confirmed_phone_number
    unless session[:phone_number_id]
      redirect_to root_path
    end
  end
end
