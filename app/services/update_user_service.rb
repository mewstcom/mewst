# typed: strict
# frozen_string_literal: true

class UpdateUserService < ApplicationService
  class Input < T::Struct
    const :user, User

    sig { params(form: Latest::UserForm).returns(Input) }
    def self.from_latest_form(form:)
      new(
        user: form.user!,
        locale: form.locale
      )
    end
  end

  class Result < T::Struct
    const :user, User
  end

  sig { params(input: Input).returns(Result) }
  def call(input:)
    form.user!.update!(
      locale: input.locale
    )

    Result.new(user: input.user)
  end
end
