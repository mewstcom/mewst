# typed: strict
# frozen_string_literal: true

class Trunk::Mutations::UpdateUser < Trunk::Mutations::Base
  argument :locale, Trunk::Types::Enums::Locale, required: false

  field :user, Trunk::Types::Objects::UserType, null: true
  field :errors, [Trunk::Types::Objects::ClientErrorType], null: false

  def resolve(locale:)
    form = Forms::User.new(locale: locale.downcase)
    form.user = context[:viewer]

    if form.invalid?
      return {
        user: nil,
        errors: form.errors.full_messages.map { |message| {message:} }
      }
    end

    result = Commands::UpdateUser.new(form:).call

    {
      user: result.user,
      errors: []
    }
  end
end
