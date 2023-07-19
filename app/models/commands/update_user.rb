# typed: strict
# frozen_string_literal: true

class Commands::UpdateUser < Commands::Base
  class Result < T::Struct
    const :user, User
  end

  sig { params(form: Forms::User).void }
  def initialize(form:)
    @form = form
  end

  sig { returns(Result) }
  def call
    form.user!.update!(form.attributes)

    Result.new(user: form.user!)
  end

  private

  sig { returns(Forms::User) }
  attr_reader :form
end
