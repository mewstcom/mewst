# typed: strict
# frozen_string_literal: true

class SignUpService < ApplicationService
  class Error < T::Struct
    const :message, String
  end

  class Result < T::Struct
    const :errors, T::Array[Error], default: []
    const :account, T.nilable(Account)
  end

  sig { params(form: SignUpForm).void }
  def initialize(form:)
    @form = form
  end

  sig { returns(Result) }
  def call
    account = Account.create!(email: @form.email)
    user = account.users.create!
    user.create_profile!(idname: @form.idname)

    Result.new(account:)
  end
end
