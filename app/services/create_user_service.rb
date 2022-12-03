# typed: strict
# frozen_string_literal: true

class CreateUserService < ApplicationService
  class Error < T::Struct
    const :message, String
  end

  class Result < T::Struct
    const :errors, T::Array[Error], default: []
    const :user, T.nilable(User)
  end

  sig { params(form: NewUserForm).void }
  def initialize(form:)
    @form = form
  end

  sig { returns(Result) }
  def call
    user = ActiveRecord::Base.transaction do
      user = User.create!
      phone_number = PhoneNumber.find_or_create_by!(value: @form.phone_number_confirmation.phone_number)
      user.create_user_phone_number!(phone_number:)
      profile = user.create_profile!(idname: @form.idname, profilable_type: :user)
      user.create_user_profile!(profile:)
      @form.phone_number_confirmation.destroy

      user
    end

    Result.new(user:)
  end
end
